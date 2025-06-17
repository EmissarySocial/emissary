package service

import (
	"encoding/hex"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

// Connection manages all interactions with the Connection collection
type Connection struct {
	collection       data.Collection
	providerService  *Provider
	keyEncryptingKey string
	host             string
}

// NewConnection returns a fully populated Connection service
func NewConnection() Connection {
	return Connection{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Connection) Refresh(collection data.Collection, providerService *Provider, keyEncryptingKey string, host string) {
	service.collection = collection
	service.providerService = providerService
	service.keyEncryptingKey = keyEncryptingKey
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Connection) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// Count returns the number of records that match the provided criteria
func (service *Connection) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

func (service *Connection) Query(criteria exp.Expression, options ...option.Option) ([]model.Connection, error) {
	result := make([]model.Connection, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Connections who match the provided criteria
func (service *Connection) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Connection from the database
func (service *Connection) Load(criteria exp.Expression, result *model.Connection) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Connection.Load", "Error loading Connection", criteria)
	}

	return nil
}

// Save adds/updates an Connection in the database
func (service *Connection) Save(connection *model.Connection, note string) error {

	const location = "service.Connection.Save"

	provider, isValidProvider := service.providerService.GetProvider(connection.ProviderID)

	if !isValidProvider {
		return derp.InternalError(location, "Invalid Provider", connection.ProviderID)
	}

	// Decode the EncryptionKey
	encryptionKey, err := hex.DecodeString(service.keyEncryptingKey)

	if err != nil {
		return derp.Wrap(err, location, "Error decoding encryption key")
	}

	// Encrypt plaintext values in vault
	if err := connection.Vault.Encrypt(encryptionKey); err != nil {
		return derp.Wrap(err, location, "Error encrypting vault values")
	}

	// Validate the value before saving
	if err := service.Schema().Validate(connection); err != nil {
		return derp.Wrap(err, location, "Error validating Connection", connection)
	}

	// Decrypt the vault data
	vault, err := service.DecryptVault(connection)

	if err != nil {
		return derp.Wrap(err, location, "Error getting vault")
	}

	switch connection.Active {

	// Connect/Update the connection
	case true:

		if err := provider.Connect(connection, vault); err != nil {
			return derp.Wrap(err, location, "Error installing connection")
		}

	// Disconnect the connection
	case false:

		if err := provider.Disconnect(connection, vault); err != nil {
			return derp.Wrap(err, location, "Error installing connection")
		}
	}

	// Save the value to the database
	if err := service.collection.Save(connection, note); err != nil {
		return derp.Wrap(err, location, "Error saving Connection", connection, note)
	}

	return nil
}

// Delete removes an Connection from the database (virtual delete)
func (service *Connection) Delete(connection *model.Connection, note string) error {

	if err := service.collection.Delete(connection, note); err != nil {
		return derp.Wrap(err, "service.Connection.Delete", "Error deleting Connection", connection, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Connection) ObjectType() string {
	return "Connection"
}

// New returns a fully initialized model.Connection as a data.Object.
func (service *Connection) ObjectNew() data.Object {
	result := model.NewConnection()
	return &result
}

func (service *Connection) ObjectID(object data.Object) primitive.ObjectID {

	if group, ok := object.(*model.Connection); ok {
		return group.ConnectionID
	}

	return primitive.NilObjectID
}

func (service *Connection) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Connection) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewConnection()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Connection) ObjectSave(object data.Object, comment string) error {
	if group, ok := object.(*model.Connection); ok {
		return service.Save(group, comment)
	}
	return derp.InternalError("service.Connection.ObjectSave", "Invalid Object Type", object)
}

func (service *Connection) ObjectDelete(object data.Object, comment string) error {
	if group, ok := object.(*model.Connection); ok {
		return service.Delete(group, comment)
	}
	return derp.InternalError("service.Connection.ObjectDelete", "Invalid Object Type", object)
}

func (service *Connection) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Connection", "Not Authorized")
}

func (service *Connection) Schema() schema.Schema {
	return schema.New(model.ConnectionSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Connection) QueryAll(options ...option.Option) ([]model.Connection, error) {
	return service.Query(exp.All(), options...)
}

func (service *Connection) QueryByType(typeID string, options ...option.Option) ([]model.Connection, error) {

	const location = "service.Connection.QueryByType"

	// RULE: typeID must not be empty
	if typeID == "" {
		return nil, derp.InternalError(location, "Invalid Type ID", typeID)
	}

	return service.Query(exp.Equal("type", typeID), options...)
}

func (service *Connection) AllAsMap() mapof.Object[model.Connection] {
	result := make(mapof.Object[model.Connection])

	if query, err := service.QueryAll(); err == nil {
		for _, connection := range query {
			result[connection.ProviderID] = connection
		}
	}

	return result
}

func (service *Connection) LoadByID(connectionID primitive.ObjectID, connection *model.Connection) error {

	const location = "service.Connection.LoadByID"

	// RULE: connectionID must not be empty
	if connectionID.IsZero() {
		return derp.InternalError(location, "Invalid Connection ID", connectionID)
	}

	criteria := exp.Equal("_id", connectionID)

	if err := service.Load(criteria, connection); err != nil {
		return derp.Wrap(err, location, "Error loading Connection", connectionID)
	}

	return nil
}

func (service *Connection) LoadByToken(token string, connection *model.Connection) error {

	const location = "service.Connection.LoadByToken"

	// Convert the token to an ObjectID
	connectionID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Connection ID", token)
	}

	// Load the Connection by ID
	return service.LoadByID(connectionID, connection)
}

func (service *Connection) LoadActiveByType(typeID string, connection *model.Connection) error {

	const location = "service.Connection.LoadActiveByType"

	// RULE: typeID must not be empty
	if typeID == "" {
		return derp.InternalError(location, "Invalid Type ID", typeID)
	}

	criteria := exp.Equal("type", typeID).AndEqual("active", true)

	if err := service.Load(criteria, connection); err != nil {
		return derp.Wrap(err, location, "Error loading Connection", typeID)
	}

	return nil
}

// LoadByProvider loads a Connection that matches the given provider.
func (service *Connection) LoadByProvider(providerID string, connection *model.Connection) error {

	const location = "service.Connection.LoadByProvider"

	// RULE: providerID must not be empty
	if providerID == "" {
		return derp.InternalError(location, "Invalid Provider ID", providerID)
	}

	criteria := exp.Equal("providerId", providerID)

	if err := service.Load(criteria, connection); err != nil {
		return derp.Wrap(err, location, "Error loading Connection", providerID)
	}

	return nil
}

// LoadOrCreateByProvider loads a Connection that matches the given provider.  If no Connection is found, a new one is created.
func (service *Connection) LoadOrCreateByProvider(providerID string) (model.Connection, error) {

	const location = "service.Connection.LoadOrCreateByProvider"

	// RULE: providerID must not be empty
	if providerID == "" {
		return model.Connection{}, derp.InternalError(location, "Invalid Provider ID", providerID)
	}

	result := model.NewConnection()

	criteria := exp.Equal("providerId", providerID)

	if err := service.Load(criteria, &result); err != nil {

		if derp.IsNotFound(err) {
			result.ProviderID = providerID
			return result, nil
		}

		return result, derp.Wrap(err, location, "Error loading Connection", providerID)
	}

	return result, nil
}

func (service *Connection) DecryptVault(connection *model.Connection) (mapof.String, error) {
	const location = "service.Connection.DecryptVault"

	// RULE: connection must not be nil
	if connection == nil {
		return nil, derp.InternalError(location, "Connection is nil")
	}

	// Decode the EncryptionKey
	encryptionKey, err := hex.DecodeString(service.keyEncryptingKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error decoding encryption key")
	}

	// Decrypt the vault
	result, err := connection.Vault.Decrypt(encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error decrypting vault")
	}

	return result, nil
}

func (service *Connection) GetAccessToken(connection *model.Connection) (oauth2.Token, error) {

	const location = "service.Connection.GetAccessToken"

	// If the token is valid, then return it immediately
	if connection.Token.Valid() {
		return *connection.Token, nil
	}

	// Fall through means that we need to refresh the access token

	// Find the correct provider...
	provider, isValidProvider := service.providerService.GetProvider(connection.ProviderID)

	if !isValidProvider {
		return oauth2.Token{}, derp.InternalError(location, "Invalid Provider", connection.ProviderID)
	}

	// Decrypt the vault (access keys will be in here)
	vault, err := service.DecryptVault(connection)

	if err != nil {
		return oauth2.Token{}, derp.Wrap(err, location, "Error decrypting vault", connection.ProviderID)
	}

	// Refresh the Access Token according to the provider's rules
	if err := provider.Refresh(connection, vault); err != nil {
		return oauth2.Token{}, derp.Wrap(err, location, "Error refreshing access token", connection.ProviderID)
	}

	// Triumphantly return the access token
	return *connection.Token, nil
}
