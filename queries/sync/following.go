package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// followingUniqueIndex names the index that enforces one active Following per (User, profileUrl).
// Its presence doubles as proof that the collection has already been de-duplicated. It is a NEW
// name (the old non-unique idx_Following_User_Profile is dropped by indexer.Sync) so that the
// dedup guard below is not fooled into skipping the scan by the pre-existing index.
const followingUniqueIndex = "idx_Following_User_Profile_Unique"

func Following(ctx context.Context, database *mongo.Database) error {

	const location = "queries.sync.Following"

	log.Trace().Str("database", database.Name()).Str("collection", "Following").Msg("COLLECTION:")

	collection := database.Collection("Following")

	// Clear out duplicate follows that would violate the unique index below. A failure here is
	// reported but not fatal: only the unique index fails to build (and is retried on the next
	// startup), while the query indexes still sync.
	if err := deduplicateFollowing(ctx, collection); err != nil {
		derp.Report(derp.Wrap(err, location, "Removing duplicate Following records", database.Name()))
	}

	return indexer.Sync(ctx, collection, indexer.IndexSet{

		// idx_Following_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Following_Recycle": recycleIndex(),

		"idx_Following_User_Folder": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "folderId", Value: 1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{
				"deleteDate": 0,
			}),
		},

		// Enforces one active Following per (userId, profileUrl) AND serves LoadByURL. The partial
		// filter scopes uniqueness to live rows with a RESOLVED profileUrl: the create path inserts
		// a row before Connect resolves the canonical actor id, and those transient empty-profileUrl
		// rows must not collide with each other. Once resolved, service.Following dedups them.
		followingUniqueIndex: mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "profileUrl", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.D{
				{Key: "deleteDate", Value: 0},
				{Key: "profileUrl", Value: bson.D{{Key: "$gt", Value: ""}}},
			}),
		},

		"idx_Following_NextPoll": mongo.IndexModel{
			Keys: bson.D{
				{Key: "nextPoll", Value: 1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{
				"deleteDate": 0,
			}),
		},
	})
}

// followingRecord is the slice of a Following that deduplicateFollowing needs in order to choose
// which of a User's competing follows of one actor survives.
type followingRecord struct {
	FollowingID primitive.ObjectID `bson:"followingId"`
	CreateDate  int64              `bson:"createDate"`
}

// followingGroup collects every active Following that one User holds against one profileUrl.
type followingGroup struct {
	Followings []followingRecord `bson:"followings"`
}

// deduplicateFollowing removes duplicate active follows -- keeping the oldest of each (User,
// profileUrl) set -- so that followingUniqueIndex can be built on a database that predates it.
// Rows are deleted directly rather than through service.Following.Delete: cleanup is local-only
// (the surviving record still carries the relationship, and no ActivityPub Undo is owed for a
// duplicate the remote never distinguished).
//
// This is a one-time migration. Once the unique index exists, duplicates can no longer be
// written, so later startups skip the scan instead of re-reading the whole collection.
func deduplicateFollowing(ctx context.Context, collection *mongo.Collection) error {

	const location = "queries.sync.deduplicateFollowing"

	// The index is only ever created after a successful cleanup, so its presence proves that
	// there is nothing left to find.
	indexed, err := indexExists(ctx, collection, followingUniqueIndex)

	if err != nil {
		return derp.Wrap(err, location, "Looking for the unique index")
	}

	if indexed {
		return nil
	}

	// Collect every (User, profileUrl) carrying more than one active follow. A resolved
	// profileUrl is required: unresolved rows are excluded from the unique index and cannot
	// collide. A lone follow cannot duplicate anything, so most rows drop out at the last stage.
	pipeline := []bson.M{
		{"$match": bson.M{"deleteDate": 0, "profileUrl": bson.M{"$gt": ""}}},
		{"$group": bson.M{
			"_id": bson.M{"userId": "$userId", "profileUrl": "$profileUrl"},
			"followings": bson.M{"$push": bson.M{
				"followingId": "$_id",
				"createDate":  "$createDate",
			}},
		}},
		{"$match": bson.M{"followings.1": bson.M{"$exists": true}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))

	if err != nil {
		return derp.Wrap(err, location, "Grouping Following records by User and profileUrl")
	}

	groups := make([]followingGroup, 0)

	if err := cursor.All(ctx, &groups); err != nil {
		return derp.Wrap(err, location, "Reading Following records from cursor")
	}

	// Work out which of the competing follows are displaced (everything but the oldest survivor).
	displaced := make([]primitive.ObjectID, 0)

	for _, group := range groups {
		displaced = append(displaced, group.displacedFollowingIDs()...)
	}

	if len(displaced) == 0 {
		return nil
	}

	// Remove them in a single round-trip.
	if _, err := collection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": displaced}}); err != nil {
		return derp.Wrap(err, location, "Deleting duplicate Following records")
	}

	log.Info().
		Str("database", collection.Database().Name()).
		Int("count", len(displaced)).
		Msg("Removed duplicate Following records")

	return nil
}

// displacedFollowingIDs returns the ID of every follow in this group except the oldest, which
// survives. Which one survives does not matter to correctness; that exactly one survives does.
func (group followingGroup) displacedFollowingIDs() []primitive.ObjectID {

	survivor := oldestFollowingID(group.Followings)

	result := make([]primitive.ObjectID, 0, len(group.Followings)-1)

	for _, record := range group.Followings {
		if record.FollowingID != survivor {
			result = append(result, record.FollowingID)
		}
	}

	return result
}

// oldestFollowingID returns the ID of the earliest-created record in a non-empty group.
func oldestFollowingID(records []followingRecord) primitive.ObjectID {

	oldest := records[0]

	for _, record := range records[1:] {
		if record.CreateDate < oldest.CreateDate {
			oldest = record
		}
	}

	return oldest.FollowingID
}
