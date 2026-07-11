package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mention records track hyperlinks from external sources to internal objects.
 * They are created from ActivityPub "Mention" tags.
 *
 * Golang RegExp syntax:
 * - https://pkg.go.dev/regexp/syntax
 * - https://github.com/google/re2/wiki/Syntax
 *
 ******************************************/

// Mention defines a service that manages Mention records
type Mention struct {
}

// NewMention returns a fully initialized Mention service
func NewMention() Mention {
	return Mention{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Mention) Refresh(factory *Factory) {
	// Nothing to refresh.
}

// Close stops any background processes controlled by this service
func (service *Mention) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Mention) collection(session data.Session) data.Collection {
	return session.Collection("Mention")
}

// Count returns the number of records that match the provided criteria
func (service *Mention) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Mentions that match the provided criteria
func (service *Mention) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Mention, error) {
	result := make([]model.Mention, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Mentions that match the provided criteria
func (service *Mention) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Mention records that match the provided criteria
func (service *Mention) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Mention], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Mention.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewMention), nil
}

// Load retrieves an Mention from the database
func (service *Mention) Load(session data.Session, criteria exp.Expression, mention *model.Mention) error {

	if err := service.collection(session).Load(notDeleted(criteria), mention); err != nil {
		return derp.Wrap(err, "service.Mention.Load", "Unable to load Mention", criteria)
	}

	return nil
}

// Save adds/updates an Mention in the database
func (service *Mention) Save(session data.Session, mention *model.Mention, note string) error {

	// Validate the value before saving
	if _, err := service.Schema().Validate(mention); err != nil {
		return derp.Wrap(err, "service.Mention.Save", "Unable to validate Mention", mention)
	}

	// Save the value to the database
	if err := service.collection(session).Save(mention, note); err != nil {
		return derp.Wrap(err, "service.Mention.Save", "Unable to save Mention", mention, note)
	}

	return nil
}

// Delete removes an Mention from the database (virtual delete)
func (service *Mention) Delete(session data.Session, mention *model.Mention, note string) error {

	criteria := exp.Equal("_id", mention.MentionID)

	// Delete this Mention
	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Mention.Delete", "Unable to delete Mention", criteria)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Mention) ObjectType() string {
	return "Mention"
}

// New returns a fully initialized model.Mention as a data.Object.
func (service *Mention) ObjectNew() data.Object {
	result := model.NewMention()
	return &result
}

func (service *Mention) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Mention); ok {
		return mention.MentionID
	}

	return primitive.NilObjectID
}

func (service *Mention) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Mention) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewMention()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Mention) ObjectSave(session data.Session, object data.Object, comment string) error {
	if mention, ok := object.(*model.Mention); ok {
		return service.Save(session, mention, comment)
	}
	return derp.Internal("service.Mention.ObjectSave", "Invalid Object Type", object)
}

func (service *Mention) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if mention, ok := object.(*model.Mention); ok {
		return service.Delete(session, mention, comment)
	}
	return derp.Internal("service.Mention.ObjectDelete", "Invalid Object Type", object)
}

func (service *Mention) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Mention", "Not Authorized")
}

func (service *Mention) Schema() schema.Schema {
	return schema.New(model.MentionSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Mention) LoadByID(session data.Session, objectType string, objectID primitive.ObjectID, mentionID primitive.ObjectID, result *model.Mention) error {
	criteria := exp.Equal("type", objectType).
		AndEqual("objectId", objectID).
		AndEqual("_id", mentionID)

	return service.Load(session, criteria, result)
}

// LoadByOrigin loads an existing Mention by its type/objectID/origin URL
func (service *Mention) LoadByOrigin(session data.Session, objectType string, objectID primitive.ObjectID, originURL string, result *model.Mention) error {

	criteria := exp.Equal("type", objectType).
		AndEqual("objectId", objectID).
		AndEqual("origin.url", originURL)

	return service.Load(session, criteria, result)
}

// LoadOrCreate loads an existing Mention or creates a new one if it doesn't exist
func (service *Mention) LoadOrCreate(session data.Session, objectType string, objectID primitive.ObjectID, originURL string) (model.Mention, error) {

	result := model.NewMention()
	err := service.LoadByOrigin(session, objectType, objectID, originURL, &result)

	// No error means the record was found
	if err == nil {
		return result, nil
	}

	// NotFound error means we should create a new record
	if derp.IsNotFound(err) {
		result.Type = objectType
		result.ObjectID = objectID
		result.Origin.URL = originURL
		return result, nil
	}

	// Other error is bad.  Return the error
	return result, derp.Wrap(err, "service.Mention.LoadOrCreate", "Unable to load Mention", objectType, objectID, originURL)
}

func (service *Mention) QueryByObjectID(session data.Session, objectType string, objectID primitive.ObjectID, options ...option.Option) ([]model.Mention, error) {
	return service.Query(session, exp.Equal("objectId", objectID).AndEqual("type", objectType), options...)
}

func (service *Mention) RangeByObjectID(session data.Session, objectType string, objectID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Mention], error) {
	return service.Range(session, exp.Equal("objectId", objectID).AndEqual("type", objectType), options...)
}

// DeleteByUserID deletes all Mentions owned by the provided UserID
func (service *Mention) DeleteByObjectID(session data.Session, objectType string, objectID primitive.ObjectID, note string) error {

	const location = "service.Mention.DeleteByUserID"

	// Retrieve all Mentions
	mentions, err := service.RangeByObjectID(session, objectType, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to query Mentions by UserID", objectType, objectID)
	}

	// Delete each mention
	for mention := range mentions {
		if err := service.Delete(session, &mention, note); err != nil {
			return derp.Wrap(err, location, "Unable to delete Mention", mention)
		}
	}

	// Success
	return nil
}
