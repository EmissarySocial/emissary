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

// Outbox manages all interactions with a user's Outbox
type Outbox struct {
	collection data.Collection
}

// NewOutbox returns a fully populated Outbox service
func NewOutbox(collection data.Collection) Outbox {
	service := Outbox{
		collection: collection,
	}

	service.Refresh(collection)
	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Outbox) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Outbox) Close() {

}

/*******************************************
 * Common Data Methods
 *******************************************/

// New creates a newly initialized Outbox that is ready to use
func (service *Outbox) New() model.Activity {
	return model.NewActivity()
}

// List returns an iterator containing all of the Outboxs who match the provided criteria
func (service *Outbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Outbox from the database
func (service *Outbox) Load(criteria exp.Expression, result *model.Activity) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error loading Outbox", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox) Save(activity *model.Activity, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(activity); err != nil {
		return derp.Wrap(err, "service.Outbox.Save", "Error cleaning Outbox", activity)
	}

	if err := service.collection.Save(activity, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error saving Outbox", activity, note)
	}

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(activity *model.Activity, note string) error {

	// Delete Outbox record last.
	if err := service.collection.Delete(activity, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error deleting Outbox", activity, note)
	}

	return nil
}

/*******************************************
 * Generic Data Methods
 *******************************************/

// ObjectType returns the type of object that this service manages
func (service *Outbox) ObjectType() string {
	return "Activity"
}

// New returns a fully initialized model.Stream as a data.Object.
func (service *Outbox) ObjectNew() data.Object {
	result := model.NewActivity()
	return &result
}

func (service *Outbox) ObjectID(object data.Object) primitive.ObjectID {

	if activity, ok := object.(*model.Activity); ok {
		return activity.ActivityID
	}

	return primitive.NilObjectID
}
func (service *Outbox) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Outbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewActivity()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Outbox) ObjectSave(object data.Object, note string) error {
	if activity, ok := object.(*model.Activity); ok {
		return service.Save(activity, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectSave", "Invalid Object Type", object)
}

func (service *Outbox) ObjectDelete(object data.Object, note string) error {
	if activity, ok := object.(*model.Activity); ok {
		return service.Delete(activity, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectDelete", "Invalid Object Type", object)
}

func (service *Outbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Inbox", "Not Authorized")
}

func (service *Outbox) Schema() schema.Schema {
	return schema.New(model.ActivitySchema())
}

/*******************************************
 * Custom Query Methods
 *******************************************/

func (service *Outbox) LoadItemByID(userID primitive.ObjectID, outboxItemID primitive.ObjectID, result *model.Activity) error {

	criteria := exp.Equal("_id", outboxItemID).AndEqual("userId", userID)

	if err := service.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error loading Activity", criteria)
	}

	return nil
}
