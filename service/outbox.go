package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox manages all Outbox records for a User.  This includes Outbox and Outbox
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

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Outbox) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Outbox) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// New creates a newly initialized Outbox that is ready to use
func (service *Outbox) New() model.OutboxMessage {
	return model.NewOutboxMessage()
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Outbox) Query(criteria exp.Expression, options ...option.Option) ([]model.OutboxMessage, error) {
	result := make([]model.OutboxMessage, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *Outbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Outbox from the database
func (service *Outbox) Load(criteria exp.Expression, result *model.OutboxMessage) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error loading Outbox", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox) Save(outboxMessage *model.OutboxMessage, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(outboxMessage); err != nil {
		return derp.Wrap(err, "service.Outbox.Save", "Error cleaning Outbox", outboxMessage)
	}

	// Calculate the rank for this outboxMessage, using the number of outboxMessages with an identical PublishDate
	if err := service.CalculateRank(outboxMessage); err != nil {
		return derp.Wrap(err, "service.Outbox.Save", "Error calculating rank", outboxMessage)
	}

	// Save the value to the database
	if err := service.collection.Save(outboxMessage, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error saving Outbox", outboxMessage, note)
	}

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(outboxMessage *model.OutboxMessage, note string) error {

	// Delete Outbox record last.
	if err := service.collection.Delete(outboxMessage, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error deleting Outbox", outboxMessage, note)
	}

	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *Outbox) DeleteMany(criteria exp.Expression, note string) error {

	it, err := service.List(criteria)

	if err != nil {
		return derp.Wrap(err, "service.Message.Delete", "Error listing streams to delete", criteria)
	}

	outboxMessage := model.NewOutboxMessage()

	for it.Next(&outboxMessage) {
		if err := service.Delete(&outboxMessage, note); err != nil {
			return derp.Wrap(err, "service.Message.Delete", "Error deleting outboxMessage", outboxMessage)
		}
		outboxMessage = model.NewOutboxMessage()
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Outbox) ObjectType() string {
	return "Outbox"
}

// New returns a fully initialized model.Stream as a data.Object.
func (service *Outbox) ObjectNew() data.Object {
	result := model.NewOutboxMessage()
	return &result
}

func (service *Outbox) ObjectID(object data.Object) primitive.ObjectID {

	if outboxMessage, ok := object.(*model.OutboxMessage); ok {
		return outboxMessage.OutboxMessageID
	}

	return primitive.NilObjectID
}

func (service *Outbox) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Outbox) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Outbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewOutboxMessage()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Outbox) ObjectSave(object data.Object, note string) error {
	if outboxMessage, ok := object.(*model.OutboxMessage); ok {
		return service.Save(outboxMessage, note)
	}
	return derp.NewInternalError("service.Outbox.ObjectSave", "Invalid Object Type", object)
}

func (service *Outbox) ObjectDelete(object data.Object, note string) error {
	if outboxMessage, ok := object.(*model.OutboxMessage); ok {
		return service.Delete(outboxMessage, note)
	}
	return derp.NewInternalError("service.Outbox.ObjectDelete", "Invalid Object Type", object)
}

func (service *Outbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Outbox", "Not Authorized")
}

func (service *Outbox) Schema() schema.Schema {
	return schema.New(model.OutboxMessageSchema())
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (service *Outbox) QueryByUserID(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.OutboxMessage, error) {
	criteria = exp.Equal("userId", userID).And(criteria)
	return service.Query(criteria, options...)
}

func (service *Outbox) LoadByID(userID primitive.ObjectID, outboxOutboxMessageID primitive.ObjectID, result *model.OutboxMessage) error {
	criteria := exp.Equal("userId", userID).
		AndEqual("_id", outboxOutboxMessageID)

	return service.Load(criteria, result)
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *Outbox) CalculateRank(outboxMessage *model.OutboxMessage) error {

	count, err := queries.CountOutboxMessages(service.collection, outboxMessage.UserID, outboxMessage.CreateDate)

	if err != nil {
		return derp.Wrap(err, "service.Outbox", "Error calculating rank", outboxMessage)
	}

	outboxMessage.Rank = (outboxMessage.CreateDate * 1000) + int64(count)
	return nil
}
