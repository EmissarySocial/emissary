package service

import (
	"iter"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Inbox manages all Inbox records for a User.
type Inbox struct {
	activityService *ActivityStream
	host            string
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

	// Save the value to the database
	if err := service.collection(session).Save(inboxActivity, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Inbox activity", inboxActivity, note)
	}

	// (async) guarantee the activity.Object is loaded into the ActivityStream cache
	go service.cacheObject(inboxActivity)

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

func (service *Inbox) LoadByID(session data.Session, userID primitive.ObjectID, messageID primitive.ObjectID, result *model.InboxActivity) error {
	criteria := exp.Equal("_id", messageID).AndEqual("userId", userID)
	return service.Load(session, criteria, result)
}

// CountByUser returns the number of InboxActivities that belong to a user
func (service *Inbox) CountByUser(session data.Session, userID primitive.ObjectID) (int64, error) {
	return service.Count(session, exp.Equal("userId", userID))
}

// RangeByUser returns a Go 1.23 RangeFunc that iterates over the InboxActivities that belong to a user (in natural chronological order)
func (service *Inbox) RangeByUser(session data.Session, userID primitive.ObjectID, startAfter string) (iter.Seq[model.InboxActivity], error) {

	// Build the base criteria
	var criteria exp.Expression = exp.Equal("userId", userID)

	// Add the "startAfter" criteria (if applicable)
	if startAfterID := service.parseMessageID(startAfter, userID); !startAfterID.IsZero() {
		criteria = criteria.AndGreaterThan("_id", startAfterID)
	}

	// Return the filtered range
	return service.Range(session, criteria, option.SortAsc("_id"))
}

/******************************************
 * Collection Interface
 ******************************************/

// CollectionID returns the identifier function for this collection
func (service *Inbox) CollectionID(userID primitive.ObjectID) collection.IdentifierFunc {
	return func() string {
		return service.host + "/@" + userID.Hex() + "/pub/inbox"
	}
}

// CollectionCount returns the counter function for this collection
func (service *Inbox) CollectionCount(session data.Session, userID primitive.ObjectID) collection.CounterFunc {
	return func() (int64, error) {
		return service.CountByUser(session, userID)
	}
}

// CollectionIterator returns the iterator function for this collection
func (service *Inbox) CollectionIterator(session data.Session, userID primitive.ObjectID) collection.IteratorFunc {

	const location = "service.Inbox.CollectionIterator"

	return func(startAfter string) (iter.Seq[mapof.Any], error) {

		result, err := service.RangeByUser(session, userID, startAfter)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unable to create iterator", "userID", userID.Hex())
		}

		return ranges.Map(result, func(item model.InboxActivity) mapof.Any {
			return item.GetJSONLD()
		}), nil
	}
}

// parseMessageID extracts the messageID from the provided URL
func (service *Inbox) parseMessageID(url string, userID primitive.ObjectID) primitive.ObjectID {

	if url == "" {
		return primitive.NilObjectID
	}

	if url == "START" {
		return primitive.NilObjectID
	}

	prefix := service.host + "/@" + userID.Hex() + "/pub/mls/messages/"

	if !strings.HasPrefix(url, prefix) {
		return primitive.NilObjectID
	}

	messageIDHex := strings.TrimPrefix(url, prefix)
	messageID, err := primitive.ObjectIDFromHex(messageIDHex)

	if err != nil {
		return primitive.NilObjectID
	}

	return messageID
}
