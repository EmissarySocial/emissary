package service

import (
	"iter"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox manages all Outbox records for a User.  This includes Outbox and Outbox
type Outbox struct {
	factory           *Factory
	followerService   *Follower
	identityService   *Identity
	importItemService *ImportItem
	ruleService       *Rule
	streamService     *Stream
	templateService   *Template
	userService       *User
	domainEmail       *DomainEmail
	lock              *sync.Mutex
	queue             *queue.Queue
	host              string
}

// NewOutbox returns a fully populated Outbox service
func NewOutbox(factory *Factory) Outbox {
	return Outbox{
		factory: factory,
		lock:    &sync.Mutex{},
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Outbox) Refresh(followerService *Follower, identityService *Identity, importItemService *ImportItem, ruleService *Rule, streamService *Stream, templateService *Template, userService *User, domainEmail *DomainEmail, queue *queue.Queue, host string) {
	service.followerService = followerService
	service.identityService = identityService
	service.importItemService = importItemService
	service.ruleService = ruleService
	service.streamService = streamService
	service.templateService = templateService
	service.userService = userService
	service.domainEmail = domainEmail
	service.queue = queue
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Outbox) Close() {
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Outbox) collection(session data.Session) data.Collection {
	return session.Collection("Outbox")
}

// New creates a newly initialized Outbox that is ready to use
func (service *Outbox) New() model.OutboxMessage {
	return model.NewOutboxMessage()
}

// Count returns the number of records that match the provided criteria
func (service *Outbox) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Outbox) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.OutboxMessage, error) {
	result := make([]model.OutboxMessage, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *Outbox) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the OutboxMessage records that match the provided criteria
func (service *Outbox) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.OutboxMessage], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Outbox.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewOutboxMessage), nil
}

// Load retrieves an Outbox from the database
func (service *Outbox) Load(session data.Session, criteria exp.Expression, result *model.OutboxMessage) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox.Load", "Unable to load Outbox message", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox) Save(session data.Session, outboxMessage *model.OutboxMessage, note string) error {

	const location = "service.Outbox.Save"

	// Calculate the ActivityURL for this message
	outboxMessage.ActivityURL = service.calcActivityURL(outboxMessage)

	// Save the value to the database
	if err := service.collection(session).Save(outboxMessage, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Outbox message", outboxMessage, note)
	}

	// (async) guarantee the message.Object is loaded into the ActivityStream cache
	go service.cacheMessage(outboxMessage)

	return nil
}

func (service *Outbox) cacheMessage(outboxMessage *model.OutboxMessage) {
	time.Sleep(1 * time.Second)
	activityService := service.factory.ActivityStream(outboxMessage.ActorType, outboxMessage.ActorID)
	_, err := activityService.Client().Load(outboxMessage.ObjectID, ascache.WithWriteOnly())
	derp.Report(err)

}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Outbox) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Folder record, without applying any additional business rules
func (service *Outbox) HardDeleteByID(session data.Session, userID primitive.ObjectID, outboxMessageID primitive.ObjectID) error {

	const location = "service.Outbox.HardDeleteByID"

	criteria := exp.Equal("actorId", userID).AndEqual("_id", outboxMessageID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Outbox Message", "userID: "+userID.Hex(), "outboxMessageID: "+outboxMessageID.Hex())
	}

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(session data.Session, outboxMessage *model.OutboxMessage, note string) error {

	const location = "service.Outbox.Delete"

	// Delete the message from the outbox
	criteria := exp.Equal("_id", outboxMessage.OutboxMessageID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Outbox message", outboxMessage, note)
	}

	// Delete the document from the cache
	activityService := service.factory.ActivityStream(outboxMessage.ActorType, outboxMessage.ActorID)
	if err := activityService.Delete(outboxMessage.ObjectID); err != nil {
		return derp.Wrap(err, location, "Unable to delete ActivityStream", outboxMessage, note)
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

func (service *Outbox) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Outbox) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewOutboxMessage()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Outbox) ObjectSave(session data.Session, object data.Object, note string) error {

	if message, ok := object.(*model.OutboxMessage); ok {
		return service.Save(session, message, note)
	}

	return derp.Internal("service.Outbox.ObjectSave", "Invalid object type", object)
}

func (service *Outbox) ObjectDelete(session data.Session, object data.Object, note string) error {

	if message, ok := object.(*model.OutboxMessage); ok {
		return service.Delete(session, message, note)
	}

	return derp.Internal("service.Outbox.ObjectDelete", "Invalid object type", object)
}

func (service *Outbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.OutboxMessage", "Not Authorized")
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
func (service *Outbox) RangeByParentID(session data.Session, actorType string, actorID primitive.ObjectID) (iter.Seq[model.OutboxMessage], error) {
	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID)

	return service.Range(session, criteria)
}

func (service *Outbox) QueryByParentAndDate(session data.Session, actorType string, actorID primitive.ObjectID, permissions model.Permissions, maxDate int64, maxRows int) ([]model.OutboxMessage, error) {

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

	if err := service.collection(session).Query(&result, criteria, options...); err != nil {
		return nil, derp.Wrap(err, location, "Unable to query outbox", actorID, maxDate)
	}

	return result, nil
}

func (service *Outbox) RangeByObjectID(session data.Session, actorType string, actorID primitive.ObjectID, objectID string) (iter.Seq[model.OutboxMessage], error) {

	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID).
		AndEqual("objectId", objectID)

	return service.Range(session, criteria)
}

func (service *Outbox) LoadByID(session data.Session, userID primitive.ObjectID, outboxMessageID primitive.ObjectID, outboxMessage *model.OutboxMessage) error {
	criteria := exp.Equal("actorId", userID).AndEqual("_id", outboxMessageID)
	return service.Load(session, criteria, outboxMessage)
}

func (service *Outbox) DeleteByParentID(session data.Session, actorType string, actorID primitive.ObjectID) error {

	const location = "service.Outbox.DeleteByParent"

	// Get all messages in this Outbox
	rangeFunc, err := service.RangeByParentID(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to query Outbox messages", actorType, actorID)
	}

	for message := range rangeFunc {
		if err := service.Delete(session, &message, "Deleted"); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to delete Outbox message", message))
		}
	}

	return nil
}

func (service *Outbox) calcActivityURL(outboxMessage *model.OutboxMessage) string {

	switch outboxMessage.ActorType {

	case model.ActorTypeApplication:
		return service.host + "/@application/pub/outbox/" + outboxMessage.OutboxMessageID.Hex()

	case model.ActorTypeSearchDomain:
		return service.host + "/@search/pub/outbox/" + outboxMessage.OutboxMessageID.Hex()

	case model.ActorTypeSearchQuery:
		return service.host + "/@search_" + outboxMessage.ActorID.Hex() + "/pub/outbox/" + outboxMessage.OutboxMessageID.Hex()

	case model.ActorTypeStream:
		return service.host + "/" + outboxMessage.ActorID.Hex() + "/pub/outbox/" + outboxMessage.OutboxMessageID.Hex()

	case model.ActorTypeUser:
		return service.host + "/@" + outboxMessage.ActorID.Hex() + "/pub/outbox/" + outboxMessage.OutboxMessageID.Hex()

	default:
		return ""
	}
}
