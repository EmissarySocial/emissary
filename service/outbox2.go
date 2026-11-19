package service

import (
	"iter"
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/floodgate"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox2 manages the "OutboxItem" collection: the ActivityPub-native outbox of local Users and Streams
type Outbox2 struct {
	// Outbox2 is replacing the legacy Outbox service -- see emissary-specs/projects/OUTBOX2-OMNIBUS.md.

	inboxService     *Inbox
	activityService  *ActivityStream
	followerService  *Follower
	circleService    *Circle
	privilegeService *Privilege
	identityService  *Identity
	ruleService      *Rule
	domainEmail      followerEmailer
	legacyOutbox     legacyOutboxRemover
	importedObjects  importedObjectFinder
	objectCache      documentDeleter
	locator          *Locator
	getSendLocator   func(data.Session) SendLocator
	queue            *queue.Queue
	floodgate        func() floodgate.Gate
	host             string
}

// NewOutbox2 returns a fully populated Outbox2 service
func NewOutbox2() Outbox2 {
	return Outbox2{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Outbox2) Refresh(factory *Factory) {
	service.inboxService = factory.Inbox()
	service.activityService = factory.ActivityStream()
	service.followerService = factory.Follower()
	service.circleService = factory.Circle()
	service.privilegeService = factory.Privilege()
	service.floodgate = factory.Floodgate
	service.identityService = factory.Identity()
	service.ruleService = factory.Rule()
	service.domainEmail = factory.Email()
	service.legacyOutbox = factory.Outbox()
	service.importedObjects = importedStreamFinder{importItemService: factory.ImportItem(), streamService: factory.Stream()}
	service.objectCache = factory.ActivityStream()
	service.locator = factory.Locator()
	service.queue = factory.Queue()
	service.host = factory.Host()
	service.getSendLocator = factory.SendLocator
}

// Close stops any background processes controlled by this service
func (service *Outbox2) Close() {
	// No background processes to stop
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the mongo collection where OutboxItems are stored
func (service *Outbox2) collection(session data.Session) data.Collection {
	return session.Collection("OutboxItem")
}

// New creates a newly initialized OutboxItem that is ready to use
func (service *Outbox2) New() model.OutboxItem {
	return model.NewOutboxItem()
}

// Count returns the number of records that match the provided criteria
func (service *Outbox2) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the OutboxItems that match the provided criteria
func (service *Outbox2) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.OutboxItem, error) {
	result := make([]model.OutboxItem, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// Range returns a Go 1.23 RangeFunc that iterates over the OutboxItems that match the provided criteria
func (service *Outbox2) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.OutboxItem], error) {

	iter, err := service.collection(session).Iterator(notDeleted(criteria), options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Outbox2.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewOutboxItem), nil
}

// Load retrieves a single OutboxItem from the database
func (service *Outbox2) Load(session data.Session, criteria exp.Expression, result *model.OutboxItem) error {

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox2.Load", "Loading Outbox activity", criteria)
	}

	return nil
}

// Save stores an OutboxItem in the database, and sends nothing. Only Publish and an import call it.
func (service *Outbox2) Save(session data.Session, item *model.OutboxItem, note string) error {

	const location = "service.Outbox2.Save"

	if err := service.collection(session).Save(item, note); err != nil {
		return derp.Wrap(err, location, "Saving Outbox activity", "activityId: "+item.ActivityID.Hex(), note)
	}

	return nil
}

// Send delivers an ActivityPub activity through the sender pipeline WITHOUT writing an
// outbox record. Use it for transient, idempotent activities (profile Updates and the like)
// that don't belong in an actor's outbox collection. The activity must carry its own actor
// and addressing (to/cc); delivery is enqueued post-commit, so a rolled-back transaction
// sends nothing. See PROFILE-UPDATE-FEDERATION.md D-1.
func (service *Outbox2) Send(session data.Session, activity mapof.Any) error {

	const location = "service.Outbox2.Send"

	document := streams.NewDocument(activity)

	// RULE: The activity must identify its actor -- the send consumer signs with this actor's key
	if document.ActorID() == "" {
		return derp.Internal(location, "Activity must include an actor", "activityId: "+document.ID())
	}

	// Queue delivery to everyone the activity names (released only after the transaction commits)
	if err := service.queueDelivery(session, document); err != nil {
		return derp.Wrap(err, location, "Queueing delivery", "activityId: "+document.ID())
	}

	// boom.
	return nil
}

// PutUserActivity publishes a User's own addressed (non-public) activity from their outbox, and
// returns the activity's server-assigned URL.
func (service *Outbox2) PutUserActivity(session data.Session, userID primitive.ObjectID, activity streams.Document) (string, error) {

	const location = "service.Outbox2.PutUserActivity"

	// RULE: Public posts are Streams; only addressed activities are delivered from here
	if activity.IsPublic() {
		return "", derp.NotImplemented(location, "Public notes are not supported at this time.")
	}

	// RULE: A User may only send activities as themselves
	actorID, err := service.locator.ParseUser(activity.ActorID())

	if err != nil {
		return "", derp.Wrap(err, location, "Parsing userID from actorID", "actorID: "+activity.ActorID(), activity.Map())
	}

	if actorID != userID {
		return "", derp.Forbidden(location, "Actor does not match authenticated user", "actorID: "+activity.ActorID(), activity.Map())
	}

	// RULE: An Undo may only reverse the actor's own activities (ActivityPub §6.10)
	if err := requireUndoActorMatches(activity); err != nil {
		return "", derp.Wrap(err, location, "Validating Undo")
	}

	// RULE: The server assigns the activity's canonical id, replacing any client-provided
	// value (ActivityPub §6.1). Client-generated ids (e.g. "urn:uuid:...") are not
	// dereferenceable URLs, and remote servers reject activities that carry them.
	activity.SetProperty(vocab.PropertyID, "")

	// RULE: The JSON-LD spelling goes too. Publish reads `@id` when `id` is empty, so a client
	// could otherwise choose its activity's URL, on this host or on someone else's.
	delete(activity.Map(), vocab.PropertyID_Alternate)

	// Publish with no permissions, so delivery follows the client's addressing alone
	activityURL, err := service.Publish(session, model.ActorTypeUser, userID, activity, model.NewPermissions())

	if err != nil {
		return "", derp.Wrap(err, location, "Publishing activity")
	}

	// RULE: conversations-mls reads the KeyPackages it addresses to its own actor within the same
	// request, so only a client's own activity is also written straight into its actor's inbox
	if err := service.loopback(session, userID, activity); err != nil {
		return "", derp.Wrap(err, location, "Saving activity to the actor's inbox", activityURL)
	}

	return activityURL, nil
}

// requireUndoActorMatches returns an error when an Undo names an activity performed by another actor
func requireUndoActorMatches(activity streams.Document) error {

	const location = "service.requireUndoActorMatches"

	// This function is only needed for `Undo` activities
	if activity.Types().NotContains(vocab.ActivityTypeUndo) {
		return nil
	}

	// RULE: Every embedded activity must name the correct actor.
	actorID := activity.ActorID()

	if actorID == "" {
		return derp.Forbidden(location, "Undo actor must not be empty")
	}

	for object := range activity.Object().Range() {

		if actorID != object.ActorID() {
			return derp.Forbidden(location, "Undo actor must match the actor of the undone activity")
		}
	}

	// Success-berg
	return nil
}

// Publish stores an activity in an actor's outbox and queues its delivery for after the transaction
// commits. It returns the activity's URL. Each writer addresses its own activity; Publish adds no one.
func (service *Outbox2) Publish(session data.Session, actorType string, actorID primitive.ObjectID, activity streams.Document, permissions model.Permissions) (string, error) {

	const location = "service.Outbox2.Publish"

	// RULE: The activity must identify its actor, before anything is stored. Delivery signs
	// with that actor's key, and expands only that actor's collections.
	if activity.ActorID() == "" {
		return "", derp.Internal(location, "Activity must include an actor", "actorType: "+actorType, "actorId: "+actorID.Hex())
	}

	item := model.NewOutboxItem()

	// Keep a carried-in canonical id (a Like or a Block has a record of its own), or mint one
	activityURL := activity.ID()

	if activityURL == "" {
		activityURL = service.locator.ActivityURL(actorType, actorID, item.ActivityID)

	} else {

		// RULE: Publish an activity once. Look its URL up first, because a duplicate-key
		// error would abort the whole request transaction.
		count, err := service.collection(session).Count(exp.Equal("url", activityURL))

		if err != nil {
			return "", derp.Wrap(err, location, "Checking for a published activity", activityURL)
		}

		if count > 0 {
			return activityURL, nil
		}
	}

	// RULE: Every activity needs an id that peers can dereference
	if activityURL == "" {
		return "", derp.Internal(location, "Cannot mint an activity URL for this actor type", "actorType: "+actorType)
	}

	// Stamp the id, so the caller and every recipient see the same URL
	activity.SetProperty(vocab.PropertyID, activityURL)

	// Describe the activity, so later lookups by URL, object, and type find it
	item.ActorType = actorType
	item.ActorID = actorID
	item.URL = activityURL
	item.Activity = activity.Map()
	item.ActivityType = activity.Type()
	item.ObjectID = activity.Object().ID()
	item.Permissions = permissions
	item.CalcRecipients()

	// Store the activity
	if err := service.Save(session, &item, "Published"); err != nil {
		return "", derp.Wrap(err, location, "Saving Outbox2 activity", activityURL)
	}

	// Deliver it once the transaction commits
	if err := service.queueDelivery(session, activity); err != nil {
		return "", derp.Wrap(err, location, "Queueing delivery", activityURL)
	}

	return activityURL, nil
}

// loopback writes a User's self-addressed activity straight into their own inbox. The delivered
// copy still arrives later, and the inbox drops it as a duplicate.
func (service *Outbox2) loopback(session data.Session, userID primitive.ObjectID, asActivity streams.Document) error {

	const location = "service.Outbox2.loopback"

	// RULE: Only an activity addressed to its own actor loops back
	item := model.NewOutboxItem()
	item.Activity = asActivity.Map()
	item.CalcRecipients()

	if !item.Recipients.Contains(service.locator.UserURL(userID)) {
		return nil
	}

	inboxActivity := model.NewInboxActivity()
	inboxActivity.ActivityID = asActivity.ID()
	inboxActivity.UserID = userID
	inboxActivity.ActorID = asActivity.ActorID()
	inboxActivity.Context = asActivity.Context()
	inboxActivity.ActivityType = asActivity.Type()
	inboxActivity.ObjectType = asActivity.Object().Type()
	inboxActivity.ObjectID = asActivity.Object().ID()
	inboxActivity.MediaType = asActivity.Object().MediaType()
	inboxActivity.ReceivedDate = time.Now().UnixMilli()
	inboxActivity.RawActivity = item.Activity
	inboxActivity.IsPublic = asActivity.IsPublic()
	inboxActivity.PublishedDate = asActivity.Published().UnixMilli()

	if err := service.inboxService.Save(session, &inboxActivity, "Saved directly from outbox"); err != nil {
		return derp.Wrap(err, location, "Saving item to inbox", "activityId: "+inboxActivity.ActivityID)
	}

	return nil
}

// queueDelivery queues the fan-out of an activity to everyone it names, for after the
// transaction commits. The task carries the activity as it was given, blind recipients included.
func (service *Outbox2) queueDelivery(session data.Session, activity streams.Document) error {

	const location = "service.Outbox2.queueDelivery"

	// RULE: The actor is read as a reference, because an activity may embed its actor as an
	// object, and a worker given no actor expands no followers and applies no one's rules.
	actorID := activity.ActorID()

	if actorID == "" {
		return derp.Internal(location, "Activity must include an actor", "activityId: "+activity.ID())
	}

	// "hostname" and "actor" are for the consumer that routes the task to this domain
	postcommit.Publish(session, service.queue, delivery.TaskFanOut, mapof.Any{
		"hostname": uri.Hostname(service.host),
		"actor":    actorID,
		"activity": activity.Map(),
	})

	// Queued, and waiting on a commit
	return nil
}

// Delete removes an OutboxItem from the database (HARD delete)
func (service *Outbox2) Delete(session data.Session, item *model.OutboxItem, note string) error {

	const location = "service.Outbox2.Delete"

	// Delete the message from the outbox
	criteria := exp.Equal("_id", item.ActivityID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting Outbox activity", item, note)
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// Schema returns the rosetta schema that describes a Outbox2
func (service *Outbox2) Schema() schema.Schema {
	return schema.New(model.OutboxItemSchema())
}

/******************************************
 * Custom Query Methods
 ******************************************/

// RangeByUser returns a Go 1.23 RangeFunc that iterates over the OutboxItems for a specific User
func (service *Outbox2) RangeByUser(session data.Session, userID primitive.ObjectID, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeUser, userID, options...)
}

// RangeByStream returns a Go 1.23 RangeFunc that iterates over the OutboxItems for a specific Stream / Content Actor
func (service *Outbox2) RangeByStream(session data.Session, streamID primitive.ObjectID, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeStream, streamID, options...)
}

// RangeBySearchQuery returns a Go 1.23 RangeFunc that iterates over the OutboxItems for a specific SearchQuery
func (service *Outbox2) RangeBySearchQuery(session data.Session, searchQueryID primitive.ObjectID, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeSearchQuery, searchQueryID, options...)
}

// RangeBySearchDomain returns a Go 1.23 RangeFunc that iterates over the OutboxItems for the global @search actor
func (service *Outbox2) RangeBySearchDomain(session data.Session, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeSearchDomain, primitive.NilObjectID, options...)
}

// RangeByApplication returns a Go 1.23 RangeFunc that iterates over the OutboxItems for the global @application actor
func (service *Outbox2) RangeByApplication(session data.Session, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	return service.RangeByActor(session, model.ActorTypeApplication, primitive.NilObjectID, options...)
}

// RangeByActor returns a Go 1.23 RangeFunc that iterates over the OutboxItems for a specific actor (actorType, actorID)
func (service *Outbox2) RangeByActor(session data.Session, actorType string, actorID primitive.ObjectID, options ...option.Option) (iter.Seq[model.OutboxItem], error) {
	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID)

	return service.Range(session, criteria, options...)
}

// QueryByParentAndDate returns one page of an actor's outbox created before maxDate, newest first, filtered by the caller's permissions
func (service *Outbox2) QueryByParentAndDate(session data.Session, actorType string, actorID primitive.ObjectID, permissions model.Permissions, maxDate int64, maxRows int) ([]model.OutboxItem, error) {

	const location = "service.Outbox2.QueryByParentAndDate"

	// RULE: A caller with no permissions sees nothing. A nil list would also send `$in: null`, which MongoDB rejects.
	if len(permissions) == 0 {
		return make([]model.OutboxItem, 0), nil
	}

	criteria := visibleCriteria(actorType, actorID, permissions).
		And(exp.LessThan("createDate", maxDate))

	result, err := service.Query(session, criteria, option.SortDesc("createDate"), option.MaxRows(int64(maxRows)))

	if err != nil {
		return nil, derp.Wrap(err, location, "Querying outbox", "actorType: "+actorType, "actorId: "+actorID.Hex())
	}

	return result, nil
}

// PagingSummary returns how many items in an actor's outbox the caller's permissions can see, and the
// maxDate that makes QueryByParentAndDate return the last page of them
func (service *Outbox2) PagingSummary(session data.Session, actorType string, actorID primitive.ObjectID, permissions model.Permissions, pageSize int) (int64, int64, error) {

	const location = "service.Outbox2.PagingSummary"

	// RULE: A page holds at least one item
	if pageSize < 1 {
		return 0, 0, derp.Internal(location, "Page size must be positive", pageSize)
	}

	// Count what the caller can see
	totalItems, err := service.CountByParent(session, actorType, actorID, permissions)

	if err != nil {
		return 0, 0, derp.Wrap(err, location, "Counting outbox", "actorType: "+actorType, "actorId: "+actorID.Hex())
	}

	// RULE: When every item fits on one page, the first page is also the last
	if totalItems <= int64(pageSize) {
		return totalItems, math.MaxInt64, nil
	}

	// The last page holds the oldest items, so its cursor is the date of the item just newer than them.
	// Reading those few dates oldest first reaches it without skipping through the whole outbox.
	lastSize := lastPageSize(totalItems, pageSize)
	criteria := visibleCriteria(actorType, actorID, permissions)
	oldest, err := service.Query(session, criteria, option.SortAsc("createDate"), option.MaxRows(lastSize+1), option.Fields("createDate"))

	if err != nil {
		return 0, 0, derp.Wrap(err, location, "Querying oldest outbox items", "actorType: "+actorType, "actorId: "+actorID.Hex())
	}

	// RULE: An outbox that shrank after it was counted pages from the start
	if int64(len(oldest)) <= lastSize {
		return totalItems, math.MaxInt64, nil
	}

	return totalItems, oldest[lastSize].CreateDate, nil
}

// CountByParent returns the number of items in an actor's outbox that the caller's permissions can see
func (service *Outbox2) CountByParent(session data.Session, actorType string, actorID primitive.ObjectID, permissions model.Permissions) (int64, error) {

	const location = "service.Outbox2.CountByParent"

	// RULE: A caller with no permissions sees nothing. A nil list would also send `$in: null`, which MongoDB rejects.
	if len(permissions) == 0 {
		return 0, nil
	}

	// RULE: Count only what the caller could read, so totalItems does not reveal private activity
	count, err := service.Count(session, visibleCriteria(actorType, actorID, permissions))

	if err != nil {
		return 0, derp.Wrap(err, location, "Counting outbox", "actorType: "+actorType, "actorId: "+actorID.Hex())
	}

	return count, nil
}

// lastPageSize returns how many items the last page holds when totalItems are split into pages of pageSize
func lastPageSize(totalItems int64, pageSize int) int64 {

	// RULE: No items, or no pages, leave nothing for a last page
	if totalItems < 1 || pageSize < 1 {
		return 0
	}

	// A remainder of zero means the last page is full
	if remainder := totalItems % int64(pageSize); remainder > 0 {
		return remainder
	}

	return int64(pageSize)
}

// visibleCriteria matches the items in an actor's outbox that a caller holding the provided permissions may read
func visibleCriteria(actorType string, actorID primitive.ObjectID, permissions model.Permissions) exp.Expression {

	// RULE: An item is visible only to a caller who holds one of its permissions
	return exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID).
		AndIn("permissions", permissions)
}

// RangeByObjectID returns an iterator over every OutboxItem in an actor's outbox that acts on the provided
// object, including an Announce of an activity that does
func (service *Outbox2) RangeByObjectID(session data.Session, actorType string, actorID primitive.ObjectID, objectID string) (iter.Seq[model.OutboxItem], error) {

	// RULE: A parent Stream announces its child's Create or Update, so its row names that activity
	// as the object. Match the child inside it too, or unpublishing the child leaves it served.
	announced := exp.Equal("activityType", vocab.ActivityTypeAnnounce).
		And(exp.Or(
			exp.Equal("activity.object.object", objectID),
			exp.Equal("activity.object.object.id", objectID),
		))

	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID).
		And(exp.Or(exp.Equal("objectId", objectID), announced))

	return service.Range(session, criteria)
}

// HasObject returns TRUE if an actor's outbox holds an activity that publishes the provided object
func (service *Outbox2) HasObject(session data.Session, actorType string, actorID primitive.ObjectID, objectID string) (bool, error) {

	const location = "service.Outbox2.HasObject"

	count, err := service.Count(session, publishedObjectCriteria(actorType, actorID, objectID))

	if err != nil {
		return false, derp.Wrap(err, location, "Counting outbox activities", "actorType: "+actorType, "actorId: "+actorID.Hex(), objectID)
	}

	return count > 0, nil
}

// publishedObjectCriteria matches the activities in an actor's legacy outbox or Outbox2 that publish the provided object
func publishedObjectCriteria(actorType string, actorID primitive.ObjectID, objectID string) exp.Expression {

	// RULE: Only a Create or Update publishes an object. A Delete names the object too, but withdraws it.
	return exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID).
		AndEqual("objectId", objectID).
		AndIn("activityType", []string{vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate})
}

// RangeByActivityURL returns an iterator over every OutboxItem in an actor's outbox whose activity URL matches
func (service *Outbox2) RangeByActivityURL(session data.Session, actorType string, actorID primitive.ObjectID, activityURL string) (iter.Seq[model.OutboxItem], error) {

	criteria := exp.Equal("actorType", actorType).
		AndEqual("actorId", actorID).
		AndEqual("url", activityURL)

	return service.Range(session, criteria)
}

// RemoveByObjectID deletes every activity in an actor's outbox that acts on the provided object,
// and sends nothing
func (service *Outbox2) RemoveByObjectID(session data.Session, actorType string, actorID primitive.ObjectID, objectID string) error {

	const location = "service.Outbox2.RemoveByObjectID"

	// RULE: An empty objectID would match every activity that has no object
	if objectID == "" {
		return nil
	}

	items, err := service.RangeByObjectID(session, actorType, actorID, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Loading outbox activities", objectID)
	}

	return service.removeAll(session, items)
}

// removeByActivityURL deletes the activity in an actor's outbox that has the provided URL
func (service *Outbox2) removeByActivityURL(session data.Session, actorType string, actorID primitive.ObjectID, activityURL string) error {

	const location = "service.Outbox2.removeByActivityURL"

	// RULE: An empty URL identifies no activity
	if activityURL == "" {
		return nil
	}

	items, err := service.RangeByActivityURL(session, actorType, actorID, activityURL)

	if err != nil {
		return derp.Wrap(err, location, "Loading outbox activities", activityURL)
	}

	return service.removeAll(session, items)
}

// removeAll deletes each of the provided activities
func (service *Outbox2) removeAll(session data.Session, items iter.Seq[model.OutboxItem]) error {

	const location = "service.Outbox2.removeAll"

	for item := range items {
		if err := service.Delete(session, &item, "Un-Publishing"); err != nil {
			return derp.Wrap(err, location, "Deleting outbox activity", "activityId: "+item.ActivityID.Hex())
		}
	}

	return nil
}

// LoadByID retrieves a single OutboxItem that matches the provided actor and activityID
func (service *Outbox2) LoadByID(session data.Session, actorType string, actorID primitive.ObjectID, activityID primitive.ObjectID, item *model.OutboxItem) error {
	criteria := exp.Equal("_id", activityID).
		AndEqual("actorId", actorID).
		AndEqual("actorType", actorType)

	return service.Load(session, criteria, item)
}

// DeleteByActor removes ALL OutboxItems that belong to the provided actor
func (service *Outbox2) DeleteByActor(session data.Session, actorType string, actorID primitive.ObjectID) error {

	const location = "service.Outbox2.DeleteByActor"

	// Get all messages in this Outbox
	rangeFunc, err := service.RangeByActor(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Querying Outbox activities", actorType, actorID)
	}

	for message := range rangeFunc {
		if err := service.Delete(session, &message, "Deleted"); err != nil {
			derp.Report(derp.Wrap(err, location, "Deleting Outbox activity", message))
		}
	}

	return nil
}
