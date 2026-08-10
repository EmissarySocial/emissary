package service

import (
	"iter"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// KeyPackage defines a service that manages the MLS KeyPackages published by each User.
type KeyPackage struct {
	host string
}

// NewKeyPackage returns a fully initialized KeyPackage service
func NewKeyPackage() KeyPackage {
	return KeyPackage{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *KeyPackage) Refresh(factory *Factory) {
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *KeyPackage) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the mongodb collection where KeyPackages are stored
func (service *KeyPackage) collection(session data.Session) data.Collection {
	return session.Collection("KeyPackage")
}

// Count returns the number of records that match the provided criteria
func (service *KeyPackage) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// List returns an iterator containing all of the KeyPackages who match the provided criteria
func (service *KeyPackage) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the KeyPackages that match the provided criteria
func (service *KeyPackage) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.KeyPackage], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.KeyPackage.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewKeyPackage), nil
}

// Query returns a slice of KeyPackages that match the provided criteria
func (service *KeyPackage) Query(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.KeyPackage], error) {
	result := make([]model.KeyPackage, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Load retrieves a KeyPackage from the database
func (service *KeyPackage) Load(session data.Session, criteria exp.Expression, keyPackage *model.KeyPackage) error {

	if err := service.collection(session).Load(notDeleted(criteria), keyPackage); err != nil {
		return derp.Wrap(err, "service.KeyPackage.Load", "Loading KeyPackage", criteria)
	}

	return nil
}

// Save adds/updates a KeyPackage in the database
func (service *KeyPackage) Save(session data.Session, keyPackage *model.KeyPackage, note string) error {

	const location = "service.KeyPackage.Save"

	// Validate the value before saving
	if _, err := service.Schema().Validate(keyPackage); err != nil {
		return derp.Wrap(err, location, "Validating KeyPackage", keyPackage)
	}

	// Save the KeyPackage to the database
	if err := service.collection(session).Save(keyPackage, note); err != nil {
		return derp.Wrap(err, location, "Saving KeyPackage", keyPackage, note)
	}

	return nil
}

// Delete removes a KeyPackage from the database (virtual delete)
func (service *KeyPackage) Delete(session data.Session, keyPackage *model.KeyPackage, note string) error {

	const location = "service.KeyPackage.Delete"

	// Delete this KeyPackage
	if err := service.collection(session).Delete(keyPackage, note); err != nil {
		return derp.Wrap(err, location, "Deleting KeyPackage", keyPackage, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *KeyPackage) ObjectType() string {
	return "KeyPackage"
}

// ObjectNew returns a fully initialized model.KeyPackage as a data.Object.
func (service *KeyPackage) ObjectNew() data.Object {
	result := model.NewKeyPackage()
	return &result
}

// ObjectID returns the primary key of the provided data.Object
func (service *KeyPackage) ObjectID(object data.Object) primitive.ObjectID {

	if keyPackage, ok := object.(*model.KeyPackage); ok {
		return keyPackage.KeyPackageID
	}

	return primitive.NilObjectID
}

// ObjectQuery populates the result value with all KeyPackages that match the provided criteria
func (service *KeyPackage) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad returns the first KeyPackage that matches the provided criteria as a data.Object
func (service *KeyPackage) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewKeyPackage()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave saves the provided data.Object, which must be a model.KeyPackage
func (service *KeyPackage) ObjectSave(session data.Session, object data.Object, comment string) error {
	if keyPackage, ok := object.(*model.KeyPackage); ok {
		return service.Save(session, keyPackage, comment)
	}
	return derp.Internal("service.KeyPackage.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete deletes the provided data.Object, which must be a model.KeyPackage
func (service *KeyPackage) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if keyPackage, ok := object.(*model.KeyPackage); ok {
		return service.Delete(session, keyPackage, comment)
	}
	return derp.Internal("service.KeyPackage.ObjectDelete", "Invalid Object Type", object)
}

// ObjectUserCan always refuses generic object-level permissions for KeyPackages
func (service *KeyPackage) ObjectUserCan(_ data.Object, _ model.Authorization, _ string) error {
	return derp.Unauthorized("service.KeyPackage", "Not Authorized")
}

// Schema returns the validation schema for KeyPackage records
func (service *KeyPackage) Schema() schema.Schema {
	return schema.New(model.KeyPackageSchema())
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *KeyPackage) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

/******************************************
 * Custom Queries
 ******************************************/

// RangeByUser returns an iterator containing all KeyPackages for the specified user
func (service *KeyPackage) RangeByUser(session data.Session, userID primitive.ObjectID) (iter.Seq[model.KeyPackage], error) {
	return service.Range(session, exp.Equal("userId", userID))
}

// QueryByUser returns a slice containing all KeyPackages for the specified user
func (service *KeyPackage) QueryByUser(session data.Session, userID primitive.ObjectID) (sliceof.Object[model.KeyPackage], error) {
	return service.Query(session, exp.Equal("userId", userID))
}

// QueryIDOnlyByUser returns an iterator containing all KeyPackages for the specified user
func (service *KeyPackage) QueryIDOnlyByUser(session data.Session, userID primitive.ObjectID) (sliceof.Object[model.IDOnly], error) {
	return service.QueryIDOnly(session, exp.Equal("userId", userID))
}

// LoadByID tries to load the KeyPackage from the database.  If no key
// exists for the designated user, then a new one is generated.
func (service *KeyPackage) LoadByID(session data.Session, userID primitive.ObjectID, keyPackageID primitive.ObjectID, keyPackage *model.KeyPackage) error {
	criteria := exp.Equal("_id", keyPackageID).AndEqual("userId", userID)
	return service.Load(session, criteria, keyPackage)
}

// LoadByToken tries to load the KeyPackage from the database
// by converting the provided string into a primitive.ObjectID.
func (service *KeyPackage) LoadByToken(session data.Session, userID primitive.ObjectID, keyPackageToken string, keyPackage *model.KeyPackage) error {

	const location = "service.KeyPackage.LoadByToken"

	keyPackageID, err := primitive.ObjectIDFromHex(keyPackageToken)

	if err != nil {
		return derp.Wrap(err, location, "KeyPackageToken must be a valid ObjectID", keyPackageToken)
	}

	return service.LoadByID(session, userID, keyPackageID, keyPackage)
}

// LoadByURL tries to load the KeyPackage from the database
// by parsing the provided ActivityPub URL.
func (service *KeyPackage) LoadByURL(session data.Session, url string, keyPackage *model.KeyPackage) error {

	const location = "service.KeyPackage.LoadByURL"

	// Parse the URL to extract the UserID and KeyPackageID
	userID, keyPackageID, err := service.ParseKeyPackageURL(url)
	if err != nil {
		return derp.Wrap(err, location, "Parsing KeyPackage URL", url, derp.WithNotFound())
	}

	// Load the KeyPackage from the database
	if err := service.LoadByID(session, userID, keyPackageID, keyPackage); err != nil {
		return derp.Wrap(err, location, "Loading KeyPackage by URL", url)
	}

	return nil
}

/******************************************
 * JSONLD
 ******************************************/

// GetJSONLD returns a JSON-LD representation of this KeyPackage
func (service *KeyPackage) GetJSONLD(keyPackage *model.KeyPackage) mapof.Any {

	// The stored summary is client-authored (e.g. an EmojiKey); fall back to generic text for legacy records.
	return mapof.Any{
		vocab.AtContext: []string{
			vocab.ContextTypeActivityStreams,
			vocab.ContextTypeSocialWebMLS,
		},
		vocab.PropertyType:         []string{vocab.CoreTypeObject, vocab.ObjectTypeKeyPackage},
		vocab.PropertyID:           service.ActivityPubURL(keyPackage.UserID, keyPackage.KeyPackageID),
		vocab.PropertyAttributedTo: service.ActivityPubAttributedToURL(keyPackage.UserID),
		vocab.PropertyTo:           vocab.NamespacePublic,
		vocab.PropertySummary:      first.String(keyPackage.Summary, "A binary-encoded cryptographic key"),
		vocab.PropertyMediaType:    keyPackage.MediaType,
		vocab.PropertyEncoding:     keyPackage.Encoding,
		vocab.PropertyContent:      keyPackage.Content,
		vocab.PropertyGenerator: mapof.Any{
			vocab.PropertyID:   keyPackage.GeneratorID,
			vocab.PropertyType: vocab.ActorTypeApplication,
			vocab.PropertyName: keyPackage.GeneratorName,
		},
		vocab.PropertyPublished:      datetime.FromUnixMilli(keyPackage.CreateDate),
		vocab.PropertyMLSCiphersuite: keyPackage.Ciphersuite,
	}
}

// ActivityPubAttributedToURL returns the ActivityPubURL for the User who owns this KeyPackage
func (service *KeyPackage) ActivityPubAttributedToURL(userID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex()
}

// ActivityPubCollectionURL returns the ActivityPub ID for the collection of keyPackages
func (service *KeyPackage) ActivityPubCollectionURL(userID primitive.ObjectID) string {
	return service.ActivityPubAttributedToURL(userID) + "/pub/keyPackages"
}

// ActivityPubURL returns the ActivityPub "ObjectID" for this KeyPackage
func (service *KeyPackage) ActivityPubURL(userID primitive.ObjectID, keyPackageID primitive.ObjectID) string {
	return service.ActivityPubCollectionURL(userID) + "/" + keyPackageID.Hex()
}

/******************************************
 * Helper Methods
 ******************************************/

// ParseKeyPackageURL extracts the UserID and KeyPackageID from an ActivityPub KeyPackage URL
func (service *KeyPackage) ParseKeyPackageURL(url string) (primitive.ObjectID, primitive.ObjectID, error) {

	const location = "service.KeyPackage.ParseKeyPackageURL"

	// Split and parse the URL
	url, found := strings.CutPrefix(url, service.host+"/@")

	if !found {
		return primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "KeyPackage URL must match host name", url)
	}

	splitURL := strings.Split(url, "/")

	if len(splitURL) != 4 {
		return primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "KeyPackage URL must have path length of 3", url)
	}

	// splitURL[0] => the UserID
	// splitURL[1] => "pub"
	// splitURL[2] => "keyPackages"
	// splitURL[3] => the KeyPackageID

	if splitURL[1] != "pub" || splitURL[2] != "keyPackages" {
		return primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "KeyPackage URL must contain /pub/keyPackages", url)
	}

	userString := splitURL[0]
	keyPackageString := splitURL[3]

	// Parse the UserID
	userID, err := primitive.ObjectIDFromHex(userString)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, location, "Parsing UserID from KeyPackage URL", url)
	}

	// Parse the KeyPackageID
	keyPackageID, err := primitive.ObjectIDFromHex(keyPackageString)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, location, "Parsing KeyPackageID from KeyPackage URL", url)
	}

	// Win.
	return userID, keyPackageID, nil
}
