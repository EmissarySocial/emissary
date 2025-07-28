package service

import (
	"iter"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox manages all Outbox records for a User.  This includes Outbox and Outbox
type Outbox struct {
	collection      data.Collection
	activityService *ActivityStream
	followerService *Follower
	identityService *Identity
	ruleService     *Rule
	streamService   *Stream
	templateService *Template
	userService     *User
	domainEmail     *DomainEmail
	lock            *sync.Mutex
	queue           *queue.Queue
	hostname        string
}

// NewOutbox returns a fully populated Outbox service
func NewOutbox() Outbox {
	return Outbox{
		lock: &sync.Mutex{},
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Outbox) Refresh(collection data.Collection, activityService *ActivityStream, followerService *Follower, identityService *Identity, ruleService *Rule, streamService *Stream, templateService *Template, userService *User, domainEmail *DomainEmail, queue *queue.Queue, hostname string) {
	service.collection = collection
	service.activityService = activityService
	service.followerService = followerService
	service.identityService = identityService
	service.ruleService = ruleService
	service.streamService = streamService
	service.templateService = templateService
	service.userService = userService
	service.domainEmail = domainEmail
	service.queue = queue
	service.hostname = hostname
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

// Count returns the number of records that match the provided criteria
func (service *Outbox) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Outbox) Query(criteria exp.Expression, options ...option.Option) ([]model.OutboxMessage, error) {
	result := make([]model.OutboxMessage, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *Outbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the OutboxMessage records that match the provided criteria
func (service *Outbox) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.OutboxMessage], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Outbox.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewOutboxMessage), nil
}

// Load retrieves an Outbox from the database
func (service *Outbox) Load(criteria exp.Expression, result *model.OutboxMessage) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox.Load", "Error loading Outbox Message", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox) Save(outboxMessage *model.OutboxMessage, note string) error {

	const location = "service.Outbox.Save"

	// Save the value to the database
	if err := service.collection.Save(outboxMessage, note); err != nil {
		return derp.Wrap(err, location, "Error saving Outbox", outboxMessage, note)
	}

	// If this message has a valid URL, then try cache it into the activitystream service.
	// nolint:errcheck
	go service.activityService.Load(outboxMessage.ObjectID)

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(outboxMessage *model.OutboxMessage, note string) error {

	const location = "service.Outbox.Delete"

	// Delete the message from the outbox
	criteria := exp.Equal("_id", outboxMessage.OutboxMessageID)

	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Error deleting Outbox", outboxMessage, note)
	}

	// Delete the document from the cache
	if err := service.activityService.Delete(outboxMessage.ObjectID); err != nil {
		return derp.Wrap(err, location, "Error deleting ActivityStream", outboxMessage, note)
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Outbox) ObjectType() string {
	return "OutboxMessage"
}

// New returns a fully initialized model.Outbox record as a data.Object.
func (service *Outbox) ObjectNew() data.Object {
	result := model.NewOutboxMessage()
	return &result
}

func (service *Outbox) ObjectID(object data.Object) primitive.ObjectID {

	if message, ok := object.(*model.OutboxMessage); ok {
		return message.OutboxMessageID
	}

	return primitive.NilObjectID
}

func (service *Outbox) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Outbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewOutboxMessage()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Outbox) ObjectSave(object data.Object, note string) error {

	if message, ok := object.(*model.OutboxMessage); ok {
		return service.Save(message, note)
	}
	return derp.InternalError("service.Outbox.ObjectSave", "Invalid object type", object)
}

func (service *Outbox) ObjectDelete(object data.Object, note string) error {
	if message, ok := object.(*model.OutboxMessage); ok {
		return service.Delete(message, note)
	}
	return derp.InternalError("service.Outbox.ObjectDelete", "Invalid object type", object)
}

func (service *Outbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.OutboxMessage", "Not Authorized")
}

func (service *Outbox) Schema() schema.Schema {
	result := schema.New(model.OutboxMessageSchema())
	result.ID = "https://emissary.social/schemas/stream"
	return result
}

/******************************************
 * Custom Query Methods
 ******************************************/

// RangeByParentID returns a Go 1.23 RangeFunc that iterates over the OutboxMessage records for a specific parent (actorType, actorID)
func (service *Outbox) RangeByParentID(actorType string, actorID primitive.ObjectID) (iter.Seq[model.OutboxMessage], error) {
	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID)

	return service.Range(criteria)
}

func (service *Outbox) QueryByParentAndDate(actorType string, actorID primitive.ObjectID, permissions model.Permissions, maxDate int64, maxRows int) ([]model.OutboxMessage, error) {

	const location = "service.Outbox.QueryByParentAndDate"

	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID).
		AndIn("permissions", permissions).
		And(exp.LessThan("createDate", maxDate))

	options := []option.Option{
		option.SortDesc("createDate"),
		option.MaxRows(int64(maxRows)),
	}

	result := make([]model.OutboxMessage, 0, maxRows)

	if err := service.collection.Query(&result, criteria, options...); err != nil {
		return nil, derp.Wrap(err, location, "Error querying outbox", actorID, maxDate)
	}

	return result, nil
}

func (service *Outbox) RangeByObjectID(actorType string, actorID primitive.ObjectID, objectID string) (iter.Seq[model.OutboxMessage], error) {

	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID).
		AndEqual("objectId", objectID)

	return service.Range(criteria)
}

func (service *Outbox) DeleteByParentID(actorType string, actorID primitive.ObjectID) error {

	const location = "service.Outbox.DeleteByParent"

	// Get all messages in this Outbox
	rangeFunc, err := service.RangeByParentID(actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Error querying Outbox Messages", actorType, actorID)
	}

	for message := range rangeFunc {
		if err := service.Delete(&message, "Deleted"); err != nil {
			derp.Report(derp.Wrap(err, location, "Error deleting Outbox Message", message))
		}
	}

	return nil
}
