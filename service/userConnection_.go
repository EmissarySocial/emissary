package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserConnection manages the connections that Users make to external services on their own behalf
type UserConnection struct {
	encryptionKey string
	host          string
}

// NewUserConnection returns a fully initialized UserConnection service
func NewUserConnection() UserConnection {
	return UserConnection{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *UserConnection) Refresh(factory *Factory) {
	service.encryptionKey = factory.MasterKey()
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *UserConnection) Close() {
	// Nothing to close here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the UserConnection collection for the provided database session
func (service *UserConnection) collection(session data.Session) data.Collection {
	return session.Collection("UserConnection")
}

// Count returns the number of UserConnection records that match the provided criteria
func (service *UserConnection) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice of all the UserConnections that match the provided criteria
func (service *UserConnection) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.UserConnection, error) {
	result := make([]model.UserConnection, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the UserConnections that match the provided criteria
func (service *UserConnection) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the UserConnection records that match the provided criteria
func (service *UserConnection) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.UserConnection], error) {

	iterator, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.UserConnection.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iterator, model.NewUserConnection), nil
}

// Load retrieves a UserConnection from the database
func (service *UserConnection) Load(session data.Session, criteria exp.Expression, userConnection *model.UserConnection) error {

	if err := service.collection(session).Load(notDeleted(criteria), userConnection); err != nil {
		return derp.Wrap(err, "service.UserConnection.Load", "Loading UserConnection", criteria)
	}

	return nil
}

// Save adds/updates a UserConnection in the database
func (service *UserConnection) Save(session data.Session, userConnection *model.UserConnection, note string) error {

	const location = "service.UserConnection.Save"

	// RULE: a connection belongs to exactly one User and reaches exactly one service
	if userConnection.UserID.IsZero() {
		return derp.Internal(location, "UserID is required")
	}

	if userConnection.Type == "" {
		return derp.BadRequest(location, "Please choose a service to connect to")
	}

	// Validate the record before anything acts on it
	if _, err := service.Schema().Validate(userConnection); err != nil {
		return derp.Wrap(err, location, "Invalid connection settings")
	}

	// RULE: install or remove this connection at the remote service BEFORE writing it. A
	// credential the service rejects must never reach the database, because everything
	// downstream reads a stored connection as a working one.
	if err := service.connect(session, userConnection); err != nil {
		return derp.Wrap(err, location, "Unable to connect to "+userConnection.Label())
	}

	// Seal the Vault
	if err := service.encryptVault(userConnection); err != nil {
		return derp.Wrap(err, location, "Encrypting connection secrets")
	}

	// Save the UserConnection to the database
	if err := service.collection(session).Save(userConnection, note); err != nil {
		return derp.Wrap(err, location, "Saving UserConnection", userConnection, note)
	}

	return nil
}

// Delete removes a UserConnection from the database, along with everything it installed remotely
func (service *UserConnection) Delete(session data.Session, userConnection *model.UserConnection, note string) error {

	const location = "service.UserConnection.Delete"

	// RULE: tear down at the remote service first, but do not let a failure there strand a
	// User whose credential was already revoked -- report it and remove the record anyway.
	if err := service.disconnect(session, userConnection); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to disconnect from "+userConnection.Label(), userConnection.UserConnectionID))
	}

	// RULE: HARD delete. The record holds a credential, so a soft-deleted copy would keep it
	// readable -- and a tombstone would occupy the one (userId, type) slot forever.
	criteria := exp.Equal("_id", userConnection.UserConnectionID).AndEqual("userId", userConnection.UserID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting UserConnection", userConnection, note)
	}

	return nil
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByUser returns every connection owned by the provided User
func (service *UserConnection) QueryByUser(session data.Session, userID primitive.ObjectID) ([]model.UserConnection, error) {
	return service.Query(session, exp.Equal("userId", userID), option.SortAsc("type"))
}

// LoadByUserAndType retrieves the connection that the provided User has made to a single service
func (service *UserConnection) LoadByUserAndType(session data.Session, userID primitive.ObjectID, connectionType string, userConnection *model.UserConnection) error {

	criteria := exp.Equal("userId", userID).AndEqual("type", connectionType)

	return service.Load(session, criteria, userConnection)
}

// LoadByUserAndToken retrieves one of a User's connections by its ID
func (service *UserConnection) LoadByUserAndToken(session data.Session, userID primitive.ObjectID, token string, userConnection *model.UserConnection) error {

	const location = "service.UserConnection.LoadByUserAndToken"

	userConnectionID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.BadRequest(location, "Invalid connection ID", token)
	}

	criteria := exp.Equal("_id", userConnectionID).AndEqual("userId", userID)

	return service.Load(session, criteria, userConnection)
}

// DeleteByUserID removes every connection owned by the provided User
func (service *UserConnection) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {

	const location = "service.UserConnection.DeleteByUserID"

	userConnections, err := service.QueryByUser(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Loading connections for User", userID)
	}

	for index := range userConnections {
		if err := service.Delete(session, &userConnections[index], note); err != nil {
			return derp.Wrap(err, location, "Deleting connection", userConnections[index].UserConnectionID)
		}
	}

	return nil
}

/******************************************
 * Vault Methods
 ******************************************/

// encryptVault seals any plaintext values written into this connection's Vault
func (service *UserConnection) encryptVault(userConnection *model.UserConnection) error {

	const location = "service.UserConnection.encryptVault"

	// RULE: do not reach for the master key unless there is something to seal. A domain with
	// a missing or malformed masterKey exists in the wild (BUG-110), and asking for one
	// would turn every save on such a domain into a configuration error.
	if !userConnection.Vault.NeedsEncryption() {
		return nil
	}

	encryptionKey, err := config.DecodeMasterKey(service.encryptionKey)

	if err != nil {
		return derp.Wrap(err, location, "Decoding encryption key")
	}

	if err := userConnection.Vault.Encrypt(encryptionKey); err != nil {
		return derp.Wrap(err, location, "Encrypting vault values")
	}

	return nil
}

// DecryptVault opens the named secrets stored in this connection's Vault
func (service *UserConnection) DecryptVault(userConnection *model.UserConnection, values ...string) (mapof.String, error) {

	const location = "service.UserConnection.DecryptVault"

	// The result is secret material: never log it, never put it in an error detail, and
	// never write it back onto a model object.

	// NILCHECK: userConnection cannot be nil
	if userConnection == nil {
		return nil, derp.Internal(location, "UserConnection cannot be nil")
	}

	encryptionKey, err := config.DecodeMasterKey(service.encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Decoding encryption key")
	}

	result, err := userConnection.Vault.Decrypt(encryptionKey, values...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Decrypting vault")
	}

	return result, nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// Schema returns the JSON Schema that validates a UserConnection
func (service *UserConnection) Schema() schema.Schema {
	return schema.New(model.UserConnectionSchema())
}

// ObjectType returns the type of object that this service manages
func (service *UserConnection) ObjectType() string {
	return "UserConnection"
}

// ObjectNew returns a fully initialized model.UserConnection as a data.Object.
func (service *UserConnection) ObjectNew() data.Object {
	result := model.NewUserConnection()
	return &result
}

// ObjectID returns the unique ID of the provided UserConnection. Implements the ModelService interface.
func (service *UserConnection) ObjectID(object data.Object) primitive.ObjectID {

	if userConnection, ok := object.(*model.UserConnection); ok {
		return userConnection.UserConnectionID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every UserConnection that matches the provided criteria. Implements the ModelService interface.
func (service *UserConnection) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectList returns an iterator of UserConnections. Implements the ModelService interface.
func (service *UserConnection) ObjectList(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(session, criteria, options...)
}

// ObjectLoad retrieves a single UserConnection as a data.Object. Implements the ModelService interface.
func (service *UserConnection) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewUserConnection()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a UserConnection in the database. Implements the ModelService interface.
func (service *UserConnection) ObjectSave(session data.Session, object data.Object, comment string) error {

	if userConnection, ok := object.(*model.UserConnection); ok {
		return service.Save(session, userConnection, comment)
	}

	return derp.Internal("service.UserConnection.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete removes a UserConnection from the database. Implements the ModelService interface.
func (service *UserConnection) ObjectDelete(session data.Session, object data.Object, comment string) error {

	if userConnection, ok := object.(*model.UserConnection); ok {
		return service.Delete(session, userConnection, comment)
	}

	return derp.Internal("service.UserConnection.ObjectDelete", "Invalid Object Type", object)
}

// ObjectUserCan checks authorization. Implements the ModelService interface.
func (service *UserConnection) ObjectUserCan(_ data.Object, _ model.Authorization, _ string) error {
	return derp.Unauthorized("service.UserConnection.ObjectUserCan", "Not Authorized")
}
