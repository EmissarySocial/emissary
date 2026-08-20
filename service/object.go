package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Object manages all interactions with the Object collection
type Object struct {
	host           string
	locatorService *Locator
}

// NewObject returns a fully populated Object service
func NewObject() Object {
	return Object{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Object) Refresh(factory *Factory) {
	service.host = factory.Host()
	service.locatorService = factory.Locator()
}

// Close stops any background processes controlled by this service
func (service *Object) Close() {
	// Nothing to close
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the Object collection for the provided database session
func (service *Object) collection(session data.Session) data.Collection {
	return session.Collection("Object")
}

// Count returns the number of records that match the provided criteria
func (service *Object) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Range returns an iterator containing all of the Objects who match the provided criteria
func (service *Object) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Object], error) {

	iterator, err := service.collection(session).Iterator(notDeleted(criteria), options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Object.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iterator, model.NewObject), nil
}

// Load retrieves an Object from the database
func (service *Object) Load(session data.Session, criteria exp.Expression, result *model.Object) error {
	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Object.Load", "Loading Object", criteria)
	}

	return nil
}

// Save adds/updates an Object in the database
func (service *Object) Save(session data.Session, object *model.Object, note string) error {

	const location = "service.Object.Save"

	attributedToID := service.host + "/@" + object.UserID.Hex()
	objectID := attributedToID + "/pub/objects/" + object.ObjectID.Hex()

	object.Value.SetString(vocab.PropertyAttributedTo, attributedToID)
	object.Value.SetString(vocab.PropertyID, objectID)

	// Save the value to the database
	if err := service.collection(session).Save(object, note); err != nil {
		return derp.Wrap(err, location, "Saving Object", object, note)
	}

	return nil
}

// Delete removes an Object from the database (virtual delete)
func (service *Object) Delete(session data.Session, object *model.Object, note string) error {

	if err := service.collection(session).Delete(object, note); err != nil {
		return derp.Wrap(err, "service.Object.Delete", "Deleting Object", object, note)
	}

	return nil
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID loads a single model.Object object that matches the provided objectID
func (service *Object) LoadByID(session data.Session, userID primitive.ObjectID, objectID primitive.ObjectID, result *model.Object) error {
	criteria := exp.Equal("_id", objectID).AndEqual("userId", userID)
	return service.Load(session, criteria, result)
}

// LoadByToken loads a single model.Object object whose objectID matches the provided token string
func (service *Object) LoadByToken(session data.Session, userID primitive.ObjectID, token string, object *model.Object) error {

	objectID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Object.LoadByToken", "Invalid ObjectID", "token", token, derp.WithNotFound())
	}

	return service.LoadByID(session, userID, objectID, object)
}

// RangeByUser returns an iterator containing all Objects created by the provided userID
func (service *Object) RangeByUser(session data.Session, userID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Object], error) {
	criteria := exp.Equal("userId", userID)
	return service.Range(session, criteria, options...)
}

/******************************************
 * Other Getters
 ******************************************/

// ActivityPubURL returns the ActivityPub URL of the provided Object
func (service *Object) ActivityPubURL(userID primitive.ObjectID, objectID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex() + "/pub/objects/" + objectID.Hex()
}

/******************************************
 * Permissions
 ******************************************/

// IsAllowed returns TRUE if the provided actor may read this Object
func (service *Object) IsAllowed(object *model.Object, actorID string) bool {

	// RULE: All requests match "Public"/anonymous role
	credentials := []string{
		vocab.NamespacePublic,
		vocab.NamespaceASPublic,
		vocab.NamespaceActivityStreamsPublic,
	}

	// RULE: If we have a valid actorID, also try to match it in the permissions.
	if actorID != "" {
		credentials = append(credentials, actorID)
	}

	return object.Permissions.ContainsAny(credentials...)
}

// NotAllowed returns TRUE if the provided actor may not read this Object
func (service *Object) NotAllowed(object *model.Object, actorID string) bool {
	return !service.IsAllowed(object, actorID)
}
