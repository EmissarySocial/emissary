package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Inbox manages all Inbox records for a User.
type Inbox struct {
	activityService  *ActivityStream
	host             string
	sseUpdateChannel chan<- realtime.Message
}

// NewInbox returns a fully populated Inbox service
func NewInbox() Inbox {
	return Inbox{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Inbox) Refresh(factory *Factory) {
	service.activityService = factory.ActivityStream()
	service.host = factory.Host()
	service.sseUpdateChannel = factory.SSEUpdateChannel()
}

// Close stops any background processes controlled by this service
func (service *Inbox) Close() {
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Inbox) collection(session data.Session) data.Collection {
	return session.Collection("Inbox")
}

// New creates a newly initialized Inbox that is ready to use
func (service *Inbox) New() model.InboxActivity {
	return model.NewInboxActivity()
}

// Count returns the number of records that match the provided criteria
func (service *Inbox) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Activities that match the provided criteria
func (service *Inbox) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.InboxActivity, error) {
	result := make([]model.InboxActivity, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Activities that match the provided criteria
func (service *Inbox) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the InboxActivity records that match the provided criteria
func (service *Inbox) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.InboxActivity], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Inbox.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewInboxActivity), nil
}

// Load retrieves an Inbox from the database
func (service *Inbox) Load(session data.Session, criteria exp.Expression, result *model.InboxActivity) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Inbox.Load", "Unable to load Inbox activity", criteria)
	}

	return nil
}

// Save adds/updates an Inbox in the database
func (service *Inbox) Save(session data.Session, inboxActivity *model.InboxActivity, note string) error {

	const location = "service.Inbox.Save"

	// RULE: InboxActivity must have an ActivityID
	if inboxActivity.ActivityID == "" {
		inboxActivity.ActivityID = "uri:uuid:" + primitive.NewObjectID().Hex()
	}

	// RULE: InboxActivity must have a UserID
	if inboxActivity.InboxActivityID.IsZero() {
		return derp.BadRequest(location, "InboxActivity.InboxActivityID must not be zero")
	}

	// RULE: InboxActivity must have a UserID
	if inboxActivity.UserID.IsZero() {
		return derp.BadRequest(location, "InboxActivity.UserID must not be zero")
	}

	// Validate the record using the schema
	if err := service.Schema().Validate(inboxActivity); err != nil {
		return derp.Wrap(err, location, "InboxActivity is invalid", inboxActivity)
	}

	// Check to see if this is a new record
	if err := service.createOrUpdate(session, inboxActivity, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Inbox activity", inboxActivity, note)
	}

	// (async) guarantee the activity.Object is loaded into the ActivityStream cache
	go service.cacheObject(inboxActivity)

	// Send realtime SSE messages to any listeners
	go service.sendSSEUpdate(inboxActivity)

	return nil
}

func (service *Inbox) createOrUpdate(session data.Session, inboxActivity *model.InboxActivity, note string) error {

	const location = "service.Inbox.createOrUpdate"

	// Check to see if this is a new record
	previousValue := model.NewInboxActivity()
	if err := service.LoadByActivityID(session, inboxActivity.UserID, inboxActivity.ActivityID, &previousValue); err != nil {
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Unable to load previous InboxActivity", inboxActivity)
		}

		inboxActivity.InboxActivityID = previousValue.InboxActivityID
		inboxActivity.CreateDate = previousValue.CreateDate
	}

	// Save the value to the database
	if err := service.collection(session).Save(inboxActivity, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Inbox activity", inboxActivity, note)
	}

	return nil
}

// cacheObject attempts to load the object associated with this InboxActivity into the ActivityStream cache.
func (service *Inbox) cacheObject(inboxActivity *model.InboxActivity) {

	// If there is no ObjectID, then there's nothing to cache
	if inboxActivity.ObjectID == "" {
		return
	}

	// Get an ActivityStream client and rewrite the object into the cache
	client := service.activityService.UserClient(inboxActivity.UserID)

	if _, err := client.Load(inboxActivity.ObjectID, ascache.WithWriteOnly()); err != nil {
		derp.Report(err)
	}
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Inbox) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Folder record, without applying any additional business rules
func (service *Inbox) HardDeleteByID(session data.Session, userID primitive.ObjectID, inboxActivityID primitive.ObjectID) error {

	const location = "service.Inbox.HardDeleteByID"

	criteria := exp.Equal("actorId", userID).AndEqual("_id", inboxActivityID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Inbox activity", "userID: "+userID.Hex(), "inboxActivityID: "+inboxActivityID.Hex())
	}

	return nil
}

// Delete removes an Inbox from the database (virtual delete)
func (service *Inbox) Delete(session data.Session, inboxActivity *model.InboxActivity, note string) error {

	const location = "service.Inbox.Delete"

	// Delete the activity from the outbox
	criteria := exp.Equal("_id", inboxActivity.InboxActivityID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Inbox activity", inboxActivity, note)
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Inbox) ObjectType() string {
	return "InboxActivity"
}

// New returns a fully initialized model.Inbox record as a data.Object.
func (service *Inbox) ObjectNew() data.Object {
	result := model.NewInboxActivity()
	return &result
}

func (service *Inbox) ObjectID(object data.Object) primitive.ObjectID {

	if message, ok := object.(*model.InboxActivity); ok {
		return message.InboxActivityID
	}

	return primitive.NilObjectID
}

func (service *Inbox) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Inbox) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewInboxActivity()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Inbox) ObjectSave(session data.Session, object data.Object, note string) error {

	if message, ok := object.(*model.InboxActivity); ok {
		return service.Save(session, message, note)
	}

	return derp.Internal("service.Inbox.ObjectSave", "Invalid object type", object)
}

func (service *Inbox) ObjectDelete(session data.Session, object data.Object, note string) error {

	if message, ok := object.(*model.InboxActivity); ok {
		return service.Delete(session, message, note)
	}

	return derp.Internal("service.Inbox.ObjectDelete", "Invalid object type", object)
}

func (service *Inbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.InboxActivity", "Not Authorized")
}

func (service *Inbox) Schema() schema.Schema {
	result := schema.New(model.InboxActivitySchema())
	result.ID = "https://emissary.social/schemas/stream"
	return result
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Inbox) LoadByToken(session data.Session, userID primitive.ObjectID, token string, result *model.InboxActivity) error {

	const location = "service.Inbox.LoadByToken"

	messageID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid InboxActivity ID", "token", token)
	}

	return service.LoadByID(session, userID, messageID, result)
}

// LoadByID retrieves the InboxActivity matching the provided unique identifier
func (service *Inbox) LoadByID(session data.Session, userID primitive.ObjectID, inboxActivityID primitive.ObjectID, result *model.InboxActivity) error {
	criteria := exp.Equal("_id", inboxActivityID).AndEqual("userId", userID)
	return service.Load(session, criteria, result)
}

// LoadByActivityID retrieves an InboxActivity from the database using the public "id" generated by the actor that sent the activity (e.g. "https://example.com/activities/12345")
func (service *Inbox) LoadByActivityID(session data.Session, userID primitive.ObjectID, activityID string, result *model.InboxActivity) error {
	criteria := exp.Equal("userId", userID).AndEqual("activityId", activityID)
	return service.Load(session, criteria, result)
}

// CountByUser returns the number of InboxActivities that belong to a user
func (service *Inbox) CountByUser(session data.Session, userID primitive.ObjectID, criteria exp.Expression) (int64, error) {
	criteria = criteria.AndEqual("userId", userID)
	return service.Count(session, criteria)
}

// RangeByUser returns a Go 1.23 RangeFunc that iterates over the InboxActivities that belong to a user (in natural chronological order)
func (service *Inbox) RangeByUser(session data.Session, userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) (iter.Seq[model.InboxActivity], error) {

	// Build the base criteria
	criteria = criteria.AndEqual("userId", userID)

	// Return the filtered range
	return service.Range(session, criteria, options...)
}

/******************************************
 * Realtime Updates
 ******************************************/

func (service *Inbox) sendSSEUpdate(activity *model.InboxActivity) {

	// Send an update on the "Inbox" topic for this User
	service.sseUpdateChannel <- realtime.NewMessage_InboxActivity_DirectMessage(activity.UserID, activity.String())

	// Additional rules for Direct Messages
	if !activity.IsPublic {

		// Send an update on the "DirectMessage" topic for this User
		service.sseUpdateChannel <- realtime.NewMessage_InboxActivity_DirectMessage(activity.UserID, activity.String())

		// Additional rules for MLS-encrypted messages
		if activity.MediaType == vocab.MediaTypeMLS {

			// Send an update on the "DirectMessage_MLS" topic for this User
			service.sseUpdateChannel <- realtime.NewMessage_InboxActivity_DirectMessage_MLS(activity.UserID, activity.String())
		}
	}
}

/******************************************
 * Collection Interface
 ******************************************/

// CollectionCount returns the counter function for this collection
func (service *Inbox) CollectionCount(session data.Session, userID primitive.ObjectID, criteria exp.Expression) collection.CounterFunc {
	return func() (int64, error) {
		return service.CountByUser(session, userID, criteria)
	}
}

// CollectionIterator returns the iterator function for this collection
func (service *Inbox) CollectionIterator(session data.Session, userID primitive.ObjectID, criteria exp.Expression) collection.IteratorFunc {

	const location = "service.Inbox.CollectionIterator"

	return func(startAfter string) (iter.Seq[mapof.Any], error) {

		// Add the "startAfter" criteria (if applicable)
		if startAfter != "" {
			marker := model.NewInboxActivity()
			if err := service.LoadByActivityID(session, userID, startAfter, &marker); err == nil {
				criteria = criteria.AndGreaterThan("_id", marker.InboxActivityID)
			}
		}

		// Get InboxActivitys for this User (sorted by insertion date)
		result, err := service.RangeByUser(session, userID, criteria, option.SortAsc("_id"))

		if err != nil {
			return nil, derp.Wrap(err, location, "Unable to create iterator", "userID", userID.Hex())
		}

		// Map into a range of JSON-LD objects
		return ranges.Map(result, func(item model.InboxActivity) mapof.Any {
			return item.GetJSONLD()
		}), nil
	}
}
