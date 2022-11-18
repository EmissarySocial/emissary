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
func (service *Outbox) New() model.OutboxItem {
	return model.NewOutboxItem()
}

// List returns an iterator containing all of the Outboxs who match the provided criteria
func (service *Outbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Outbox from the database
func (service *Outbox) Load(criteria exp.Expression, result *model.OutboxItem) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error loading Outbox", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox) Save(inboxItem *model.OutboxItem, note string) error {

	if err := service.collection.Save(inboxItem, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error saving Outbox", inboxItem, note)
	}

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(inboxItem *model.OutboxItem, note string) error {

	// Delete Outbox record last.
	if err := service.collection.Delete(inboxItem, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error deleting Outbox", inboxItem, note)
	}

	return nil
}

/*******************************************
 * Generic Data Methods
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Outbox) ObjectNew() data.Object {
	result := model.NewOutboxItem()
	return &result
}

func (service *Outbox) ObjectID(object data.Object) primitive.ObjectID {

	if inboxItem, ok := object.(*model.OutboxItem); ok {
		return inboxItem.OutboxItemID
	}

	return primitive.NilObjectID
}
func (service *Outbox) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Outbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewOutboxItem()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Outbox) ObjectSave(object data.Object, note string) error {
	if inboxItem, ok := object.(*model.OutboxItem); ok {
		return service.Save(inboxItem, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectSave", "Invalid Object Type", object)
}

func (service *Outbox) ObjectDelete(object data.Object, note string) error {
	if inboxItem, ok := object.(*model.OutboxItem); ok {
		return service.Delete(inboxItem, note)
	}
	return derp.NewInternalError("service.Inbox.ObjectDelete", "Invalid Object Type", object)
}

func (service *Outbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Inbox", "Not Authorized")
}

func (service *Outbox) Schema() schema.Element {
	return model.OutboxItemSchema()
}

/*******************************************
 * Custom Query Methods
 *******************************************/

func (service *Outbox) LoadItemByID(userID primitive.ObjectID, outboxItemID primitive.ObjectID, result *model.OutboxItem) error {

	criteria := exp.Equal("_id", outboxItemID).AndEqual("userId", userID)

	if err := service.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error loading OutboxItem", criteria)
	}

	return nil
}
