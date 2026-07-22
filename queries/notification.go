package queries

import (
	"context"
	"math"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// notificationUnread is the readDate sentinel for an UNREAD Notification (see model.Notification,
// where NewNotification stamps ReadDate = math.MaxInt64).  A READ notification has readDate < this.
const notificationUnread = int64(math.MaxInt64)

// NotificationOverCap pairs a User with the number of live notifications they own.  It is the
// result shape of NotificationsOverCap.
type NotificationOverCap struct {
	UserID primitive.ObjectID `bson:"_id"`
	Count  int64              `bson:"count"`
}

// NotificationsOverCap returns every User who owns more than `capacity` live (not soft-deleted)
// notifications, along with their current live count.  It is the first half of the daily per-user
// cap trim: the (normally empty) result set is exactly the users whose store must be trimmed.
func NotificationsOverCap(session data.Session, capacity int64) ([]NotificationOverCap, error) {

	const location = "queries.NotificationsOverCap"

	// Guarantee that we're using MongoDB
	collection := mongoCollection(session.Collection("Notification"))

	if collection == nil {
		return nil, derp.Internal(location, "Collection must be a MongoDB collection")
	}

	// $match live rows, $group by userId counting each, then $match only users over the cap.
	pipeline := []bson.M{
		{"$match": bson.M{"deleteDate": 0}},
		{"$group": bson.M{"_id": "$userId", "count": bson.M{"$sum": 1}}},
		{"$match": bson.M{"count": bson.M{"$gt": capacity}}},
	}

	// Set a max timeout of 180 seconds (3 minutes) to run this query
	timeout, cancel := timeoutContext(180)
	defer cancel()

	cursor, err := collection.Aggregate(timeout, pipeline)

	if err != nil {
		return nil, derp.Wrap(err, location, "Aggregating notification counts", capacity)
	}

	result := make([]NotificationOverCap, 0)

	if err := cursor.All(timeout, &result); err != nil {
		return nil, derp.Wrap(err, location, "Reading aggregation results", capacity)
	}

	return result, nil
}

// TrimNotificationsForUser hard-deletes a User's oldest notifications until at most `capacity` live
// notifications remain, deleting READ rows before UNREAD (see planNotificationTrim).  `count` is the
// User's current live-notification count (as returned by NotificationsOverCap), so this call does not
// re-count the whole store.
func TrimNotificationsForUser(session data.Session, userID primitive.ObjectID, capacity int64, count int64) error {

	const location = "queries.TrimNotificationsForUser"

	// Guarantee that we're using MongoDB
	collection := mongoCollection(session.Collection("Notification"))

	if collection == nil {
		return derp.Internal(location, "Collection must be a MongoDB collection")
	}

	// Set a max timeout of 180 seconds (3 minutes) to run this query
	timeout, cancel := timeoutContext(180)
	defer cancel()

	// Count this User's READ notifications so the plan knows how much of the surplus can be absorbed
	// without touching unread rows.
	readFilter := bson.M{"userId": userID, "deleteDate": 0, "readDate": bson.M{"$lt": notificationUnread}}

	readCount, err := collection.CountDocuments(timeout, readFilter)

	if err != nil {
		return derp.Wrap(err, location, "Counting read notifications", userID)
	}

	plan := planNotificationTrim(count, readCount, capacity)

	// Delete the oldest READ rows first...
	if err := trimOldestNotifications(timeout, collection, readFilter, plan.Read); err != nil {
		return derp.Wrap(err, location, "Trimming read notifications", userID)
	}

	// ...then the oldest UNREAD rows, only if trimming every read row was not enough.
	unreadFilter := bson.M{"userId": userID, "deleteDate": 0, "readDate": notificationUnread}

	if err := trimOldestNotifications(timeout, collection, unreadFilter, plan.Unread); err != nil {
		return derp.Wrap(err, location, "Trimming unread notifications", userID)
	}

	return nil
}

// notificationTrimPlan says how many of a User's oldest READ and UNREAD notifications to hard-delete
// to bring their live count down to the cap.
type notificationTrimPlan struct {
	Read   int64 // number of oldest READ notifications to delete
	Unread int64 // number of oldest UNREAD notifications to delete (only after every READ row is gone)
}

// planNotificationTrim computes the read-first trim plan for a User holding `count` live
// notifications, `readCount` of which are read, against the target `capacity`.  It is factored out of
// TrimNotificationsForUser so the ordering can be tested without a live database -- getting it wrong
// silently deletes the wrong notifications.
//
// RULE: READ notifications are deleted before UNREAD ones.  A junk flood that arrives already-read
// (suppressed notifications are born read -- see service.Notification.notify) is trimmed first, so it
// cannot evict the unread history the recipient actually cares about.  UNREAD rows are touched only
// when deleting every READ row is still not enough to reach the cap (i.e. the User holds more than
// `capacity` genuinely-unread rows -- itself pathological).
func planNotificationTrim(count int64, readCount int64, capacity int64) notificationTrimPlan {

	// Under (or at) the cap: nothing to trim.
	if count <= capacity {
		return notificationTrimPlan{}
	}

	surplus := count - capacity

	// The surplus fits entirely within the read rows: delete that many oldest read rows, keep every unread.
	if surplus <= readCount {
		return notificationTrimPlan{Read: surplus}
	}

	// Not enough read rows: delete them all, then take the remainder from the oldest unread rows.
	return notificationTrimPlan{
		Read:   readCount,
		Unread: surplus - readCount,
	}
}

// trimOldestNotifications hard-deletes the `deleteCount` oldest rows matching `filter` (by createDate
// ascending).  It finds the createDate of the first row to KEEP and deletes everything strictly older,
// so any createDate ties on that boundary are RETAINED -- erring toward keeping notifications rather
// than over-deleting.  When `deleteCount` reaches or exceeds the number of matching rows, every
// matching row is deleted.
func trimOldestNotifications(ctx context.Context, collection *mongo.Collection, filter bson.M, deleteCount int64) error {

	const location = "queries.trimOldestNotifications"

	// Nothing to delete for this read-state.
	if deleteCount <= 0 {
		return nil
	}

	// Find the boundary row: the (deleteCount)-th oldest matching row (0-indexed) -- the first row that
	// must be KEPT.  Skipping `deleteCount` rows lands exactly on it.
	findOptions := options.FindOne().
		SetSort(bson.D{{Key: "createDate", Value: 1}}).
		SetSkip(deleteCount).
		SetProjection(bson.M{"createDate": 1})

	var boundary struct {
		CreateDate int64 `bson:"createDate"`
	}

	err := collection.FindOne(ctx, filter, findOptions).Decode(&boundary)

	// Default to deleting every matching row (the "fewer than deleteCount rows exist" case).
	deleteFilter := filter

	switch err {

	case nil:
		// A boundary exists: delete strictly older than it.  The boundary row and its createDate ties survive.
		deleteFilter = copyFilter(filter)
		deleteFilter["createDate"] = bson.M{"$lt": boundary.CreateDate}

	case mongo.ErrNoDocuments:
		// Fewer than deleteCount matching rows exist -- delete them all (deleteFilter stays == filter).

	default:
		return derp.Wrap(err, location, "Finding trim boundary", filter, deleteCount)
	}

	if _, err := collection.DeleteMany(ctx, deleteFilter); err != nil {
		return derp.Wrap(err, location, "Deleting old notifications", deleteFilter)
	}

	return nil
}

// copyFilter returns a shallow copy of a bson.M filter, so callers can add a bound without mutating
// the original (which is reused across the count and trim queries).
func copyFilter(original bson.M) bson.M {

	result := make(bson.M, len(original)+1)

	for key, value := range original {
		result[key] = value
	}

	return result
}
