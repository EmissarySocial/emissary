package service

import (
	"iter"
	"math"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Notification defines a service that manages recipient-centric Notification records.
// Notifications are created on the inbound ActivityPub path (NotifyFromActivity) whenever
// another actor mentions, replies to, reacts to, or follows a local User.
type Notification struct {
	followingService *Following
	ruleService      *Rule
	streamService    *Stream
	userService      *User
	queue            *queue.Queue
	host             string
}

// NewNotification returns a fully initialized Notification service
func NewNotification() Notification {
	return Notification{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Notification) Refresh(factory *Factory) {
	service.followingService = factory.Following()
	service.ruleService = factory.Rule()
	service.streamService = factory.Stream()
	service.userService = factory.User()
	service.queue = factory.Queue()
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *Notification) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the Notification collection for the provided database session
func (service *Notification) collection(session data.Session) data.Collection {
	return session.Collection("Notification")
}

// Count returns the number of Notifications that match the provided criteria
func (service *Notification) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Notifications that match the provided criteria
func (service *Notification) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Notification, error) {
	result := make([]model.Notification, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Notifications that match the provided criteria
func (service *Notification) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Notifications that match the provided criteria
func (service *Notification) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Notification], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Notification.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewNotification), nil
}

// Load retrieves an Notification from the database
func (service *Notification) Load(session data.Session, criteria exp.Expression, notification *model.Notification) error {

	if err := service.collection(session).Load(notDeleted(criteria), notification); err != nil {
		return derp.Wrap(err, "service.Notification.Load", "Loading Notification", criteria)
	}

	return nil
}

// Save adds/updates an Notification in the database
func (service *Notification) Save(session data.Session, notification *model.Notification, note string) error {

	const location = "service.Notification.Save"

	// Validate the value before saving
	if _, err := service.Schema().Validate(notification); err != nil {
		return derp.Wrap(err, location, "Validating Notification", notification)
	}

	// Save the value to the database
	if err := service.collection(session).Save(notification, note); err != nil {
		return derp.Wrap(err, location, "Saving Notification", notification, note)
	}

	return nil
}

// Delete removes an Notification from the database (virtual/soft delete)
func (service *Notification) Delete(session data.Session, notification *model.Notification, note string) error {

	const location = "service.Notification.Delete"

	// Delete this Notification
	if err := service.collection(session).Delete(notification, note); err != nil {
		return derp.Wrap(err, location, "Deleting Notification", notification)
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Notification) ObjectType() string {
	return "Notification"
}

// ObjectNew returns a fully initialized model.Notification as a data.Object.
func (service *Notification) ObjectNew() data.Object {
	result := model.NewNotification()
	return &result
}

// ObjectID returns the unique ID of the provided Notification. Implements the ModelService interface.
func (service *Notification) ObjectID(object data.Object) primitive.ObjectID {

	if notification, ok := object.(*model.Notification); ok {
		return notification.NotificationID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Notification that matches the provided criteria. Implements the ModelService interface.
func (service *Notification) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Notification as a data.Object. Implements the ModelService interface.
func (service *Notification) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewNotification()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a Notification in the database. Implements the ModelService interface.
func (service *Notification) ObjectSave(session data.Session, object data.Object, note string) error {
	if notification, ok := object.(*model.Notification); ok {
		return service.Save(session, notification, note)
	}
	return derp.Internal("service.Notification.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete marks a Notification as deleted. Implements the ModelService interface.
func (service *Notification) ObjectDelete(session data.Session, object data.Object, note string) error {
	if notification, ok := object.(*model.Notification); ok {
		return service.Delete(session, notification, note)
	}
	return derp.Internal("service.Notification.ObjectDelete", "Invalid Object Type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a Notification. Implements the ModelService interface.
func (service *Notification) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Notification", "Not Authorized")
}

// Schema returns the rosetta schema that describes a Notification
func (service *Notification) Schema() schema.Schema {
	return schema.New(model.NotificationSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID loads a single Notification by its ID, scoped to the owning User.
func (service *Notification) LoadByID(session data.Session, userID primitive.ObjectID, notificationID primitive.ObjectID, notification *model.Notification) error {
	criteria := exp.Equal("_id", notificationID).AndEqual("userId", userID)
	return service.Load(session, criteria, notification)
}

// QueryByUserID returns Notifications owned by the provided User, newest first.
func (service *Notification) QueryByUserID(session data.Session, userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Notification, error) {
	criteria = exp.Equal("userId", userID).And(criteria)
	options = append(options, option.SortDesc("createDate"))
	return service.Query(session, criteria, options...)
}

// RangeByUserID iterates over every Notification owned by the provided User.
func (service *Notification) RangeByUserID(session data.Session, userID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Notification], error) {
	return service.Range(session, exp.Equal("userId", userID), options...)
}

// QueryByStreamID returns Notifications that reference a specific local Stream (stream-page display).
func (service *Notification) QueryByStreamID(session data.Session, streamID primitive.ObjectID, options ...option.Option) ([]model.Notification, error) {
	options = append(options, option.SortDesc("createDate"))
	return service.Query(session, exp.Equal("streamId", streamID), options...)
}

// HasUnread returns TRUE if the User has at least one unread Notification.  Suppressed
// notifications are saved already-read (see notify), so no channel filter is needed here —
// unread rows are surfaced rows by construction.
func (service *Notification) HasUnread(session data.Session, userID primitive.ObjectID) (bool, error) {

	const location = "service.Notification.HasUnread"

	criteria := exp.Equal("userId", userID).AndEqual("readDate", int64(math.MaxInt64))

	notification := model.NewNotification()
	err := service.Load(session, criteria, &notification)

	if err == nil {
		return true, nil
	}

	if derp.IsNotFound(err) {
		return false, nil
	}

	return false, derp.Wrap(err, location, "Loading unread Notification", userID)
}

// CountUnread returns the number of unread Notifications owned by the provided User,
// optionally filtered to specific notification types (the notifications-page tabs).
func (service *Notification) CountUnread(session data.Session, userID primitive.ObjectID, types ...string) (int64, error) {

	criteria := exp.Equal("userId", userID).AndEqual("readDate", int64(math.MaxInt64))

	if len(types) > 0 {
		criteria = criteria.And(exp.In("type", types))
	}

	return service.Count(session, criteria)
}

// LoadByActivityID loads a single Notification for a User by the AP id of its triggering activity.
func (service *Notification) LoadByActivityID(session data.Session, userID primitive.ObjectID, activityID string, notification *model.Notification) error {
	criteria := exp.Equal("userId", userID).AndEqual("activityId", activityID)
	return service.Load(session, criteria, notification)
}

// LoadOrCreate loads an existing Notification for (userID, activityID) or returns a new,
// unsaved Notification if none exists.  Used to dedup Update-vs-Create re-notification.
func (service *Notification) LoadOrCreate(session data.Session, userID primitive.ObjectID, activityID string) (model.Notification, error) {

	result := model.NewNotification()

	// Only try to dedup when we have a real activityID to match on
	if activityID != "" {
		err := service.LoadByActivityID(session, userID, activityID, &result)

		if err == nil {
			return result, nil
		}

		if !derp.IsNotFound(err) {
			return result, derp.Wrap(err, "service.Notification.LoadOrCreate", "Loading Notification", userID, activityID)
		}
	}

	// NotFound (or empty activityID) means we create a fresh record
	result.UserID = userID
	result.ActivityID = activityID
	return result, nil
}

/******************************************
 * Read/Unread State
 ******************************************/

// MarkAllRead sets the readDate on every unread Notification owned by the provided User.
// If one or more types are provided, only Notifications of those types are marked read;
// with no types, every unread Notification is marked read.
func (service *Notification) MarkAllRead(session data.Session, userID primitive.ObjectID, readDate int64, types ...string) error {

	const location = "service.Notification.MarkAllRead"

	criteria := exp.Equal("userId", userID).AndEqual("readDate", int64(math.MaxInt64)).AndEqual("deleteDate", 0)

	if len(types) > 0 {
		criteria = criteria.And(exp.In("type", types))
	}

	update := bson.M{"$set": bson.M{"readDate": readDate}}

	if err := queries.RawUpdate(session.Context(), service.collection(session), criteria, update); err != nil {
		return derp.Wrap(err, location, "Marking notifications read", userID)
	}

	return nil
}

/******************************************
 * Maintenance
 ******************************************/

// PurgeBefore hard-deletes notifications whose createDate is older than the provided cutoff
// (Unix epoch MILLISECONDS, matching journal.createDate).  Called from the daily
// PurgeNotifications task.
//
// RULE: Retention is UNIFORM -- read and unread notifications age out on the same clock.
// Read-state is deliberately not part of the criteria.  A Notification is a derived display
// snapshot (the activity itself lives on in the User's Inbox), so a stale one costs a pointer,
// not content; and because suppressed notifications are born read (see notify), a read-only
// purge would age out passive history while leaving an unread flood untouched forever.
func (service *Notification) PurgeBefore(session data.Session, cutoffMillis int64) error {

	const location = "service.Notification.PurgeBefore"

	criteria := exp.LessThan("createDate", cutoffMillis)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Purging old notifications", cutoffMillis)
	}

	return nil
}

// PurgeOverCap enforces the per-User notification ceiling: any User holding more than `capacity` live
// notifications has their oldest rows hard-deleted down to the cap, READ rows first (see
// queries.planNotificationTrim).  This is the storage backstop against a notification flood -- it
// bounds worst-case row count regardless of delivery rate or activity id, which retention alone
// cannot (a flood lands in minutes; retention purges in months).  See the NOTIFICATION-FLOOD-CONTROL
// spec.  Called from the daily PurgeNotifications task.
func (service *Notification) PurgeOverCap(session data.Session, capacity int64) error {

	const location = "service.Notification.PurgeOverCap"

	// RULE: a non-positive cap would trim EVERY user to zero (or below).  Treat it as "disabled" so a
	// misconfiguration cannot wipe the collection.
	if capacity <= 0 {
		return nil
	}

	// Find the (normally empty) set of users whose live notification count exceeds the cap.
	overCap, err := queries.NotificationsOverCap(session, capacity)

	if err != nil {
		return derp.Wrap(err, location, "Finding over-cap users", capacity)
	}

	// Trim each over-cap user back down to the cap.
	for _, user := range overCap {
		if err := queries.TrimNotificationsForUser(session, user.UserID, capacity, user.Count); err != nil {
			return derp.Wrap(err, location, "Trimming notifications", user.UserID, capacity)
		}
	}

	return nil
}

/******************************************
 * Bulk Delete Behaviors
 ******************************************/

// DeleteByUserID soft-deletes every Notification owned by the provided User.
func (service *Notification) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {

	const location = "service.Notification.DeleteByUserID"

	rangeFunc, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Querying Notifications by UserID", userID)
	}

	for notification := range rangeFunc {
		if err := service.Delete(session, &notification, note); err != nil {
			return derp.Wrap(err, location, "Deleting Notification", notification)
		}
	}

	return nil
}

// DeleteByStreamID soft-deletes every Notification that references the provided Stream.
func (service *Notification) DeleteByStreamID(session data.Session, streamID primitive.ObjectID, note string) error {

	const location = "service.Notification.DeleteByStreamID"

	notifications, err := service.QueryByStreamID(session, streamID)

	if err != nil {
		return derp.Wrap(err, location, "Querying Notifications by StreamID", streamID)
	}

	for _, notification := range notifications {
		if err := service.Delete(session, &notification, note); err != nil {
			return derp.Wrap(err, location, "Deleting Notification", notification)
		}
	}

	return nil
}

// DeleteByActivityID soft-deletes the Notification (if any) for a User matching the provided
// activityID.  Used to reverse a notification when its triggering activity is Undone/Deleted.
func (service *Notification) DeleteByActivityID(session data.Session, userID primitive.ObjectID, activityID string, note string) error {

	const location = "service.Notification.DeleteByActivityID"

	if activityID == "" {
		return nil
	}

	notification := model.NewNotification()

	if err := service.LoadByActivityID(session, userID, activityID, &notification); err != nil {

		// A notification that is already gone needs no deleting
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading Notification", userID, activityID)
	}

	if err := service.Delete(session, &notification, note); err != nil {
		return derp.Wrap(err, location, "Deleting Notification", notification)
	}

	return nil
}

// DeleteFollowByActor soft-deletes any FOLLOW notification that the given actor created for the
// provided User.  An unfollow is identified by WHO unfollowed (the actor) — NOT by the Follow
// activity's id, which is frequently absent or synthetic (see inbox_SaveActivity) and so cannot
// be matched reliably against the id referenced by a later Undo.  The actor, by contrast, is
// always present as a top-level property on the Undo, so it needs no dereferencing.
func (service *Notification) DeleteFollowByActor(session data.Session, userID primitive.ObjectID, actorURL string, note string) error {

	const location = "service.Notification.DeleteFollowByActor"

	if actorURL == "" {
		return nil
	}

	criteria := exp.Equal("userId", userID).
		AndEqual("type", model.NotificationTypeFollow).
		AndEqual("actor.profileUrl", actorURL)

	rangeFunc, err := service.Range(session, criteria)

	if err != nil {
		return derp.Wrap(err, location, "Querying FOLLOW notifications by actor", userID, actorURL)
	}

	// Delete every matching FOLLOW notification (normally one, but loop in case dedup let
	// duplicates through under a varying/synthetic activityId).
	for notification := range rangeFunc {
		if err := service.Delete(session, &notification, note); err != nil {
			return derp.Wrap(err, location, "Deleting FOLLOW notification", notification)
		}
	}

	return nil
}

// DeleteByObjectURL soft-deletes every Notification for a User whose ObjectURL matches the
// provided URL.  Used when a post is Deleted/Tombstoned to retract mention/reply notifications.
func (service *Notification) DeleteByObjectURL(session data.Session, userID primitive.ObjectID, objectURL string, note string) error {

	const location = "service.Notification.DeleteByObjectURL"

	if objectURL == "" {
		return nil
	}

	criteria := exp.Equal("userId", userID).AndEqual("objectUrl", objectURL)

	rangeFunc, err := service.Range(session, criteria)

	if err != nil {
		return derp.Wrap(err, location, "Querying Notifications by ObjectURL", userID, objectURL)
	}

	for notification := range rangeFunc {
		if err := service.Delete(session, &notification, note); err != nil {
			return derp.Wrap(err, location, "Deleting Notification", notification)
		}
	}

	return nil
}
