package service

import (
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/assanitizer"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/hannibal/property"
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
	// No background processes to stop
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the mongo collection where InboxActivities are stored
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
		return nil, derp.Wrap(err, "service.Inbox.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewInboxActivity), nil
}

// Load retrieves an Inbox from the database
func (service *Inbox) Load(session data.Session, criteria exp.Expression, result *model.InboxActivity) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Inbox.Load", "Loading Inbox activity", criteria)
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

	// RULE: AS2 §5.1 -- blind addressing MUST NOT be disclosed to any other party, and
	// RawActivity is served verbatim by InboxActivity.GetJSONLD() to the inbox collection and
	// the SSE feed. This is the persistence boundary: the last point where the document is still
	// ours alone, and the first point where it becomes readable. (BUG-07)
	//
	// Copy BEFORE stripping. RawActivity aliases a live map on both write paths -- Document.Map()
	// hands back the underlying container, and Outbox2.Save assigns OutboxItem.Activity directly
	// -- and callers upstream still read full addressing from it: IsPublic, isDirectMessage,
	// CalcRecipients, and Deliver's RangeAddressees. Stripping in place would silently drop
	// blind recipients from delivery enumeration.
	if inboxActivity.RawActivity != nil {
		rawActivity := property.Map(inboxActivity.RawActivity).Clone().Map()
		assanitizer.StripKeys(rawActivity, vocab.PropertyBTo, vocab.PropertyBCC)
		inboxActivity.RawActivity = mapof.Any(rawActivity)
	}

	// Validate the record using the schema
	if _, err := service.Schema().Validate(inboxActivity); err != nil {
		return derp.Wrap(err, location, "InboxActivity is invalid", inboxActivity)
	}

	// Check to see if this is a new record
	if err := service.createOrUpdate(session, inboxActivity, note); err != nil {
		return derp.Wrap(err, location, "Saving Inbox activity", inboxActivity, note)
	}

	// Removing this because it breaks with messages coming from Bonfire
	// (async) guarantee the activity.Object is loaded into the ActivityStream cache
	// go service.cacheObject(inboxActivity)

	// Send realtime SSE messages to any listeners
	go service.sendSSEUpdate(inboxActivity)

	return nil
}

// createOrUpdate saves an InboxActivity, reusing the identity of any earlier copy of the same activity
func (service *Inbox) createOrUpdate(session data.Session, inboxActivity *model.InboxActivity, note string) error {

	const location = "service.Inbox.createOrUpdate"

	// Check to see if this is a new record
	previousValue := model.NewInboxActivity()
	if err := service.LoadByActivityID(session, inboxActivity.UserID, inboxActivity.ActivityID, &previousValue); err == nil {
		inboxActivity.InboxActivityID = previousValue.InboxActivityID
		inboxActivity.CreateDate = previousValue.CreateDate
	} else if !derp.IsNotFound(err) {
		return derp.Wrap(err, location, "Loading previous InboxActivity", inboxActivity)
	}

	// Save the value to the database
	if err := service.collection(session).Save(inboxActivity, note); err != nil {
		return derp.Wrap(err, location, "Saving Inbox activity", inboxActivity, note)
	}

	return nil
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

// HardDeleteByID removes a specific InboxActivity record, without applying any additional business rules
func (service *Inbox) HardDeleteByID(session data.Session, userID primitive.ObjectID, inboxActivityID primitive.ObjectID) error {

	const location = "service.Inbox.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", inboxActivityID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting Inbox activity", "userID: "+userID.Hex(), "inboxActivityID: "+inboxActivityID.Hex())
	}

	return nil
}

// Delete removes an InboxActivity from the database (hard delete)
func (service *Inbox) Delete(session data.Session, inboxActivity *model.InboxActivity, note string) error {

	const location = "service.Inbox.Delete"

	// Delete the activity from the inbox
	criteria := exp.Equal("_id", inboxActivity.InboxActivityID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting Inbox activity", inboxActivity, note)
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

// ObjectNew returns a fully initialized model.InboxActivity record as a data.Object.
func (service *Inbox) ObjectNew() data.Object {
	result := model.NewInboxActivity()
	return &result
}

// ObjectID returns the primary key of the provided InboxActivity object
func (service *Inbox) ObjectID(object data.Object) primitive.ObjectID {

	if message, ok := object.(*model.InboxActivity); ok {
		return message.InboxActivityID
	}

	return primitive.NilObjectID
}

// ObjectQuery populates the result value with all InboxActivities that match the provided criteria
func (service *Inbox) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single InboxActivity that matches the provided criteria, as a data.Object
func (service *Inbox) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewInboxActivity()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave saves the provided InboxActivity object to the database
func (service *Inbox) ObjectSave(session data.Session, object data.Object, note string) error {

	if message, ok := object.(*model.InboxActivity); ok {
		return service.Save(session, message, note)
	}

	return derp.Internal("service.Inbox.ObjectSave", "Invalid object type", object)
}

// ObjectDelete removes the provided InboxActivity object from the database (virtual delete)
func (service *Inbox) ObjectDelete(session data.Session, object data.Object, note string) error {

	if message, ok := object.(*model.InboxActivity); ok {
		return service.Delete(session, message, note)
	}

	return derp.Internal("service.Inbox.ObjectDelete", "Invalid object type", object)
}

// ObjectUserCan always returns Unauthorized: InboxActivities are never edited through the generic data.Object path
func (service *Inbox) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.InboxActivity", "Not Authorized")
}

// Schema returns the validating schema for InboxActivity objects
func (service *Inbox) Schema() schema.Schema {
	result := schema.New(model.InboxActivitySchema())
	result.ID = "https://emissary.social/schemas/stream"
	return result
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByToken loads a single InboxActivity from a User's inbox, identified by its hex-encoded ID
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

// IsDuplicateActivity returns TRUE if the provided activityID has already been processed for this user (e.g. due to retries or multiple deliveries)
func (service *Inbox) IsDuplicateActivity(session data.Session, userID primitive.ObjectID, activityID string) bool {

	const location = "service.Inbox.IsDuplicateActivity"

	// If there is no activityID, then it cannot be a duplicate
	if activityID == "" {
		return false
	}

	criteria := exp.Equal("userId", userID).AndEqual("activityId", activityID)
	count, err := service.Count(session, criteria)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Checking for duplicate activity", "userID", userID, "activityID", activityID))
		return false
	}

	return count > 0
}

/******************************************
 * Realtime Updates
 ******************************************/

// sendSSEUpdate publishes an InboxActivity to the realtime channels that its visibility calls for
func (service *Inbox) sendSSEUpdate(activity *model.InboxActivity) {

	// Send an update on the "Inbox" topic for this User
	service.sseUpdateChannel <- realtime.NewMessage_InboxActivity(activity.UserID, activity.String())

	// Additional rules for Direct Messages
	if !activity.IsPublic {

		// The DM/MLS topics carry the rule labels (4C), matching what their collections serve.
		// TODO(ben): revisit once the client half lands -- the labels in SSE payloads are not
		// consumed yet, and this may be trimmable back to String() to save the work.
		payload := activity.LabeledJSON()

		// Send an update on the "DirectMessage" topic for this User
		service.sseUpdateChannel <- realtime.NewMessage_InboxActivity_DirectMessage(activity.UserID, payload)

		// Additional rules for MLS-encrypted messages
		if activity.MediaType == vocab.MediaTypeMLS {

			// Send an update on the "DirectMessage_MLS" topic for this User
			service.sseUpdateChannel <- realtime.NewMessage_InboxActivity_DirectMessage_MLS(activity.UserID, payload)
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

// inboxPageMaxRows caps each inbox page at the database. hannibal serves at most 60 items per
// collection page (collection.maxItemsPerPage) and trims client-side, so without this limit
// MongoDB materializes and sorts the user's entire inbox only to have all but 60 rows discarded.
const inboxPageMaxRows = 60

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

		// Get InboxActivitys for this User (sorted by insertion date), capped at one page
		result, err := service.RangeByUser(session, userID, criteria, option.SortAsc("_id"), option.MaxRows(inboxPageMaxRows))

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating iterator", "userID", userID.Hex())
		}

		// Map into a range of JSON-LD objects
		return ranges.Map(result, func(item model.InboxActivity) mapof.Any {
			return item.GetJSONLD()
		}), nil
	}
}

// labelChunkSize is the batching knob for CollectionIteratorWithLabels: one rules query per this
// many items. It is deliberately independent of the collection page size -- if the two drift, the
// worst case is one extra small query per page, never truncation.
const labelChunkSize = 60

// CollectionIteratorWithLabels returns a collection iterator that merges each item's rule labels
// into its JSON under the reserved "emissary:labels" key: served = persisted (the receive-time
// stamp) ∪ current (the viewer's rules right now). Current rules are queried once per chunk, and
// each item is then evaluated in memory.
func (service *Inbox) CollectionIteratorWithLabels(session data.Session, userID primitive.ObjectID, criteria exp.Expression, ruleService *Rule) collection.IteratorFunc {

	const location = "service.Inbox.CollectionIteratorWithLabels"

	return func(startAfter string) (iter.Seq[mapof.Any], error) {

		// Add the "startAfter" criteria (if applicable)
		if startAfter != "" {
			marker := model.NewInboxActivity()
			if err := service.LoadByActivityID(session, userID, startAfter, &marker); err == nil {
				criteria = criteria.AndGreaterThan("_id", marker.InboxActivityID)
			}
		}

		// Get InboxActivitys for this User (sorted by insertion date), capped at one page
		result, err := service.RangeByUser(session, userID, criteria, option.SortAsc("_id"), option.MaxRows(inboxPageMaxRows))

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating iterator", "userID", userID.Hex())
		}

		// Collect items into chunks, evaluate each chunk against the viewer's current rules, and
		// yield the labeled JSON
		return func(yield func(mapof.Any) bool) {

			now := time.Now().Unix()
			chunk := make([]model.InboxActivity, 0, labelChunkSize)

			// flush evaluates the pending chunk and yields its labeled items
			flush := func() bool {

				if len(chunk) == 0 {
					return true
				}

				rules := service.chunkRules(session, userID, chunk, ruleService)

				for _, labeled := range labeledChunkJSON(chunk, rules, now) {
					if !yield(labeled) {
						return false
					}
				}

				chunk = chunk[:0]
				return true
			}

			for item := range result {

				chunk = append(chunk, item)

				if len(chunk) >= labelChunkSize {
					if !flush() {
						return
					}
				}
			}

			_ = flush()
		}, nil
	}
}

// chunkRules queries the viewer's Rules matching any sender in the chunk: ONE indexed query for
// the whole chunk, evaluated per-item in memory afterward.
func (service *Inbox) chunkRules(session data.Session, userID primitive.ObjectID, chunk []model.InboxActivity, ruleService *Rule) []model.RuleSummary {

	const location = "service.Inbox.chunkRules"

	// Union the actor match keys across the chunk (the engine re-checks membership per item,
	// so a broader set is safe by design)
	keys := make([]string, 0, len(chunk)*2)

	for _, item := range chunk {
		keys = append(keys, model.ActorMatchKeys(item.ActorID)...)
	}

	rules, err := ruleService.QueryByMatchKeys(session, userID, keys)

	// Fail OPEN: this is a display path, so a rules blip serves persisted stamps only rather
	// than turning the whole collection into a 500
	if err != nil {
		derp.Report(derp.Wrap(err, location, "Querying rules for inbox labels; serving persisted labels only", userID))
		return nil
	}

	return rules
}

// labeledChunkJSON evaluates each activity in the chunk against the pre-fetched rules and returns
// its JSON with the merged (current ∪ persisted) labels applied. Pure: no I/O.
func labeledChunkJSON(chunk []model.InboxActivity, rules []model.RuleSummary, now int64) []mapof.Any {

	result := make([]mapof.Any, 0, len(chunk))

	for _, item := range chunk {

		// Evaluate the viewer's current rules against this sender, then merge with the
		// receive-time stamp (ties keep the current attribution -- the persisted rule may have
		// been deleted since)
		current := model.NewRuleDispositionForKeys(model.ActorMatchKeys(item.ActorID), rules, now)
		merged := current.Merge(item.Disposition)

		// Apply (or scrub) the reserved labels property on the served JSON
		raw := item.GetJSONLD()

		if raw == nil {
			raw = mapof.Any{}
		}

		merged.ApplyLabels(raw)
		result = append(result, raw)
	}

	return result
}
