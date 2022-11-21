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

// Inbox manages all interactions with a user's Inbox
type Inbox struct {
	collection data.Collection
}

// NewInbox returns a fully populated Inbox service
func NewInbox(collection data.Collection) Inbox {
	service := Inbox{
		collection: collection,
	}

	service.Refresh(collection)
	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Inbox) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Inbox) Close() {

}

/*******************************************
 * Common Data Methods
 *******************************************/

// New creates a newly initialized Inbox that is ready to use
func (service *Inbox) New() model.Activity {
	return model.NewActivity()
}

// List returns an iterator containing all of the Inboxs who match the provided criteria
func (service *Inbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Inbox from the database
func (service *Inbox) Load(criteria exp.Expression, result *model.Activity) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error loading Inbox", criteria)
	}

	return nil
}

// Save adds/updates an Inbox in the database
func (service *Inbox) Save(activity *model.Activity, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(activity); err != nil {
		return derp.Wrap(err, "service.Inbox.Save", "Error cleaning Inbox", activity)
	}

	// Save the value to the database
	if err := service.collection.Save(activity, note); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error saving Inbox", activity, note)
	}

	return nil
}

// Delete removes an Inbox from the database (virtual delete)
func (service *Inbox) Delete(activity *model.Activity, note string) error {

	// Delete Inbox record last.
	if err := service.collection.Delete(activity, note); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error deleting Inbox", activity, note)
	}

	return nil
}

/*******************************************
 * Generic Data Methods
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Inbox) ObjectNew() data.Object {
	result := model.NewActivity()
	return &result
}

func (service *Inbox) ObjectID(object data.Object) primitive.ObjectID {

	if activity, ok := object.(*model.Activity); ok {
		return activity.ActivityID
	}

	return primitive.NilObjectID
}
func (service *Inbox) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Inbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewActivity()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Inbox) ObjectSave(object data.Object, note string) error {
	if activity, ok := object.(*model.Activity); ok {
		return service.Save(activity, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectSave", "Invalid Object Type", object)
}

func (service *Inbox) ObjectDelete(object data.Object, note string) error {
	if activity, ok := object.(*model.Activity); ok {
		return service.Delete(activity, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectDelete", "Invalid Object Type", object)
}

func (service *Inbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Inbox", "Not Authorized")
}

func (service *Inbox) Schema() schema.Schema {
	return schema.New(model.ActivitySchema())
}

/*******************************************
 * Custom Query Methods
 *******************************************/

func (service *Inbox) LoadItemByID(userID primitive.ObjectID, outboxItemID primitive.ObjectID, result *model.Activity) error {

	criteria := exp.Equal("_id", outboxItemID).AndEqual("userId", userID)

	if err := service.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Inbox", "Error loading Activity", criteria)
	}

	return nil
}
