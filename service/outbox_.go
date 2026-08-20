package service

import (
	"iter"
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

// Outbox manages all Outbox records for a User.
type Outbox struct {
	activityService   *ActivityStream
	followerService   *Follower
	identityService   *Identity
	importItemService *ImportItem
	ruleService       *Rule
	streamService     *Stream
	templateService   *Template
	userService       *User
	domainEmail       *DomainEmail
	queue             *queue.Queue
	host              string
}

// NewOutbox returns a fully populated Outbox service
func NewOutbox() Outbox {
	return Outbox{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Outbox) Refresh(factory *Factory) {
	service.activityService = factory.ActivityStream()
	service.followerService = factory.Follower()
	service.identityService = factory.Identity()
	service.importItemService = factory.ImportItem()
	service.ruleService = factory.Rule()
	service.streamService = factory.Stream()
	service.templateService = factory.Template()
	service.userService = factory.User()
	service.domainEmail = factory.Email()
	service.queue = factory.Queue()
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *Outbox) Close() {
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the Outbox collection for the provided database session
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
		return nil, derp.Wrap(err, "service.Outbox.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewOutboxMessage), nil
}

// Load retrieves an Outbox from the database
func (service *Outbox) Load(session data.Session, criteria exp.Expression, result *model.OutboxMessage) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox.Load", "Loading Outbox message", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox) Save(session data.Session, outboxMessage *model.OutboxMessage, note string) error {

	const location = "service.Outbox.Save"

	// Mint an ActivityURL ONLY when the message does not already carry a canonical one. A first-class
	// activity (e.g. a Like/Dislike/Announce or a Block) arrives with its own ID already stored in
	// ActivityURL by Outbox.Publish; overwriting it here with the minted /pub/outbox/<id> form breaks
	// D7 — the CollectionItem then gets keyed by the minted URL while the Undo references the canonical
	// /pub/liked/<id> URL, so unreacts never match. See COLLECTIONS-REDESIGN.md D7.
	if outboxMessage.ActivityURL == "" {
		outboxMessage.ActivityURL = service.calcActivityURL(outboxMessage)
	}

	// Save the value to the database
	if err := service.collection(session).Save(outboxMessage, note); err != nil {
		return derp.Wrap(err, location, "Saving Outbox message", outboxMessage, note)
	}

	// (async) guarantee the message.Object is loaded into the ActivityStream cache
	go service.cacheMessage(outboxMessage)

	return nil
}

// cacheMessage warms the ActivityStream cache with a message that this server just published
func (service *Outbox) cacheMessage(outboxMessage *model.OutboxMessage) {

	// Wait for things to settle.  IDK, man
	time.Sleep(1 * time.Second)

	client := service.activityService.Client(outboxMessage.ActorType, outboxMessage.ActorID)

	if _, err := client.Load(outboxMessage.ObjectID, ascache.WithWriteOnly()); err != nil {
		derp.Report(err)
	}
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
		return derp.Wrap(err, location, "Deleting Outbox Message", "userID: "+userID.Hex(), "outboxMessageID: "+outboxMessageID.Hex())
	}

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(session data.Session, outboxMessage *model.OutboxMessage, note string) error {

	const location = "service.Outbox.Delete"

	// Delete the message from the outbox
	criteria := exp.Equal("_id", outboxMessage.OutboxMessageID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting Outbox message", outboxMessage, note)
	}

	// Delete the document from the cache
	if err := service.activityService.Delete(outboxMessage.ObjectID); err != nil {
		return derp.Wrap(err, location, "Deleting ActivityStream", outboxMessage, note)
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

// ObjectID returns the unique ID of the provided Outbox. Implements the ModelService interface.
func (service *Outbox) ObjectID(object data.Object) primitive.ObjectID {

	if message, ok := object.(*model.OutboxMessage); ok {
		return message.OutboxMessageID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Outbox that matches the provided criteria. Implements the ModelService interface.
func (service *Outbox) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Outbox as a data.Object. Implements the ModelService interface.
func (service *Outbox) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewOutboxMessage()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a Outbox in the database. Implements the ModelService interface.
func (service *Outbox) ObjectSave(session data.Session, object data.Object, note string) error {

	if message, ok := object.(*model.OutboxMessage); ok {
		return service.Save(session, message, note)
	}

	return derp.Internal("service.Outbox.ObjectSave", "Invalid object type", object)
}

// ObjectDelete marks a Outbox as deleted. Implements the ModelService interface.
func (service *Outbox) ObjectDelete(session data.Session, object data.Object, note string) error {

	if message, ok := object.(*model.OutboxMessage); ok {
		return service.Delete(session, message, note)
	}

	return derp.Internal("service.Outbox.ObjectDelete", "Invalid object type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a Outbox. Implements the ModelService interface.
func (service *Outbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.OutboxMessage", "Not Authorized")
}

// Schema returns the rosetta schema that describes a Outbox
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

// QueryByParentAndDate returns one page of an Actor's outbox, filtered by the caller's permissions
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
		return nil, derp.Wrap(err, location, "Querying outbox", actorID, maxDate)
	}

	return result, nil
}

// RangeByObjectID returns an iterator over every OutboxMessage that publishes the provided object
func (service *Outbox) RangeByObjectID(session data.Session, actorType string, actorID primitive.ObjectID, objectID string) (iter.Seq[model.OutboxMessage], error) {

	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID).
		AndEqual("objectId", objectID)

	return service.Range(session, criteria)
}

// RangeByActivityURL returns every OutboxMessage in this Actor's outbox whose canonical
// activity URL matches. Used to find-and-remove a first-class activity (e.g. the original
// Like) when it is undone — the counterpart to RangeByObjectID, which matches by the URL of
// the OBJECT an activity acted upon. See COLLECTIONS-REDESIGN.md D7.
func (service *Outbox) RangeByActivityURL(session data.Session, actorType string, actorID primitive.ObjectID, activityURL string) (iter.Seq[model.OutboxMessage], error) {

	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID).
		AndEqual("activityUrl", activityURL)

	return service.Range(session, criteria)
}

// LoadByID retrieves a single Outbox using its unique ID
func (service *Outbox) LoadByID(session data.Session, userID primitive.ObjectID, outboxMessageID primitive.ObjectID, outboxMessage *model.OutboxMessage) error {
	criteria := exp.Equal("actorId", userID).AndEqual("_id", outboxMessageID)
	return service.Load(session, criteria, outboxMessage)
}

// DeleteByParentID marks every Outbox belonging to the provided parent as deleted
func (service *Outbox) DeleteByParentID(session data.Session, actorType string, actorID primitive.ObjectID) error {

	const location = "service.Outbox.DeleteByParent"

	// Get all messages in this Outbox
	rangeFunc, err := service.RangeByParentID(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Querying Outbox messages", actorType, actorID)
	}

	for message := range rangeFunc {
		if err := service.Delete(session, &message, "Deleted"); err != nil {
			derp.Report(derp.Wrap(err, location, "Deleting Outbox message", message))
		}
	}

	return nil
}

// calcActivityURL returns the public URL of the provided OutboxMessage
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
