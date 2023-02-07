package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityStream manages all interactions with a user's ActivityStream
type ActivityStream struct {
	collection data.Collection
}

// NewActivityStream returns a fully populated ActivityStream service
func NewActivityStream(collection data.Collection) ActivityStream {
	service := ActivityStream{
		collection: collection,
	}

	service.Refresh(collection)
	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *ActivityStream) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *ActivityStream) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// New creates a newly initialized ActivityStream that is ready to use
func (service *ActivityStream) New() model.ActivityStream {
	return model.NewActivityStream(model.ActivityStreamContainerUndefined)
}

// Query returns a slice of ActivityStreams that math the provided criteria
func (service *ActivityStream) Query(criteria exp.Expression, options ...option.Option) ([]model.ActivityStream, error) {
	result := []model.ActivityStream{}
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the ActivityStreams that match the provided criteria
func (service *ActivityStream) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an ActivityStream from the database
func (service *ActivityStream) Load(criteria exp.Expression, result *model.ActivityStream) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Report(derp.Wrap(err, "service.ActivityStream.Load", "Error loading ActivityStream", criteria))
	}

	return nil
}

// Save adds/updates an ActivityStream in the database
func (service *ActivityStream) Save(activityStream *model.ActivityStream, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(activityStream); err != nil {
		return derp.Wrap(err, "service.ActivityStream.Save", "Error cleaning ActivityStream", activityStream)
	}

	// TODO: CRITICAL: How to identify duplicates?

	// Save the value to the database
	if err := service.collection.Save(activityStream, note); err != nil {
		return derp.Wrap(err, "service.ActivityStream", "Error saving ActivityStream", activityStream, note)
	}

	return nil
}

// Delete removes an ActivityStream from the database (virtual delete)
func (service *ActivityStream) Delete(activityStream *model.ActivityStream, note string) error {

	// Delete ActivityStream record last.
	if err := service.collection.Delete(activityStream, note); err != nil {
		return derp.Wrap(err, "service.ActivityStream", "Error deleting ActivityStream", activityStream, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *ActivityStream) ObjectType() string {
	return "ActivityStream"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *ActivityStream) ObjectNew() data.Object {
	result := model.NewActivityStream(model.ActivityStreamContainerUndefined)
	return &result
}

func (service *ActivityStream) ObjectID(object data.Object) primitive.ObjectID {

	if activityStream, ok := object.(*model.ActivityStream); ok {
		return activityStream.ActivityStreamID
	}

	return primitive.NilObjectID
}

func (service *ActivityStream) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *ActivityStream) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *ActivityStream) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewActivityStream(model.ActivityStreamContainerUndefined)
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *ActivityStream) ObjectSave(object data.Object, comment string) error {
	if activityStream, ok := object.(*model.ActivityStream); ok {
		return service.Save(activityStream, comment)
	}
	return derp.NewInternalError("service.ActivityStream.ObjectSave", "Invalid object type", object)
}

func (service *ActivityStream) ObjectDelete(object data.Object, comment string) error {
	if activityStream, ok := object.(*model.ActivityStream); ok {
		return service.Delete(activityStream, comment)
	}
	return derp.NewInternalError("service.ActivityStream.ObjectDelete", "Invalid object type", object)
}

func (service *ActivityStream) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.ActivityStream", "Not Authorized")
}

func (service *ActivityStream) Schema() schema.Schema {
	return schema.New(model.ActivityStreamSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *ActivityStream) ListByContainer(userID primitive.ObjectID, container model.ActivityStreamContainer, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria.AndEqual("userId", userID).AndEqual("container", container), options...)
}

func (service *ActivityStream) ListInbox(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.ListByContainer(userID, model.ActivityStreamContainerInbox, criteria, options...)
}

func (service *ActivityStream) ListOutbox(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.ListByContainer(userID, model.ActivityStreamContainerOutbox, criteria, options...)
}

func (service *ActivityStream) LoadFromContainer(userID primitive.ObjectID, container model.ActivityStreamContainer, activityStreamID primitive.ObjectID, activityStream *model.ActivityStream) error {

	criteria := exp.And(
		exp.Equal("_id", activityStreamID),
		exp.Equal("userId", userID),
		exp.Equal("container", container),
	)

	return service.Load(criteria, activityStream)
}

func (service *ActivityStream) LoadFromInbox(activityStreamID primitive.ObjectID, userID primitive.ObjectID, activityStream *model.ActivityStream) error {
	return service.LoadFromContainer(userID, model.ActivityStreamContainerInbox, activityStreamID, activityStream)
}

func (service *ActivityStream) LoadFromOutbox(activityStreamID primitive.ObjectID, userID primitive.ObjectID, activityStream *model.ActivityStream) error {
	return service.LoadFromContainer(userID, model.ActivityStreamContainerOutbox, activityStreamID, activityStream)
}
