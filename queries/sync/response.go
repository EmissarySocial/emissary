package sync

import (
	"context"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// responseUniqueIndex names the index that enforces one Response per (User, Object, type).
// Its presence doubles as proof that the collection has already been de-duplicated.
const responseUniqueIndex = "idx_Response_User_Object_Type"

// Response synchronizes the indexes on the Response collection
func Response(ctx context.Context, database *mongo.Database) error {

	const location = "queries.sync.Response"

	log.Trace().Str("database", database.Name()).Str("collection", "Response").Msg("COLLECTION:")

	collection := database.Collection("Response")

	// Clear out any reactions that would violate the unique index below.  A failure here is
	// reported but not fatal: only the unique index fails to build (and is retried on the next
	// startup), while the query indexes still sync.
	if err := deduplicateResponses(ctx, collection); err != nil {
		derp.Report(derp.Wrap(err, location, "Removing duplicate Responses", database.Name()))
	}

	return indexer.Sync(ctx, collection, indexer.IndexSet{

		// idx_Response_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Response_Recycle": recycleIndex(),

		"idx_Response_User": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "type", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},

		"idx_Response_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "object", Value: 1},
				{Key: "type", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},

		// Enforces one Response per (userId, object, type).  This makes SetResponse
		// concurrency-safe: it inserts optimistically and relies on this index to reject a
		// racing duplicate of the same reaction, which it then reads as a success.
		//
		// The partial filter scopes uniqueness to live rows (deleteDate == 0), mirroring
		// service.notDeleted().  Responses are hard-deleted today, so this guards against
		// legacy soft-deleted rows rather than anything the current code writes.
		//
		// NOTE: this index cannot span the Like/Dislike contradiction, because those are two
		// different `type` values.  Mutual exclusivity is enforced in service.SetResponse.
		responseUniqueIndex: mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "object", Value: 1},
				{Key: "type", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.D{
				{Key: "deleteDate", Value: 0},
			}),
		},
	})
}

// responseRecord is the slice of a Response that deduplicateResponses needs in order to choose
// between a User's competing reactions.
type responseRecord struct {
	ResponseID primitive.ObjectID `bson:"responseId"`
	Type       string             `bson:"type"`
	CreateDate int64              `bson:"createDate"`
}

// responseGroup collects every reaction that one User made on one Object.
type responseGroup struct {
	Responses []responseRecord `bson:"responses"`
}

// deduplicateResponses removes duplicate and contradictory reactions -- keeping only the newest
// of each -- so that responseUniqueIndex can be built on a database that predates it.  Rows are
// deleted directly rather than through service.Response.Delete, because the handlers that wrote
// them bypassed the publishing funnel entirely: nothing was ever federated, so no Undo is owed.
//
// This is a one-time migration.  Once the unique index exists, duplicates can no longer be
// written, so later startups skip the scan instead of re-reading the whole collection.
func deduplicateResponses(ctx context.Context, collection *mongo.Collection) error {

	const location = "queries.sync.deduplicateResponses"

	// The index is only ever created after a successful cleanup, so its presence proves that
	// there is nothing left to find.
	indexed, err := indexExists(ctx, collection, responseUniqueIndex)

	if err != nil {
		return derp.Wrap(err, location, "Looking for the unique index")
	}

	if indexed {
		return nil
	}

	// Collect every (User, Object) carrying more than one reaction.  A lone reaction cannot
	// contradict anything, so the overwhelming majority of rows drop out here.
	pipeline := []bson.M{
		{"$match": bson.M{"deleteDate": 0}},
		{"$group": bson.M{
			"_id": bson.M{"userId": "$userId", "object": "$object"},
			"responses": bson.M{"$push": bson.M{
				"responseId": "$_id",
				"type":       "$type",
				"createDate": "$createDate",
			}},
		}},
		{"$match": bson.M{"responses.1": bson.M{"$exists": true}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))

	if err != nil {
		return derp.Wrap(err, location, "Grouping Responses by User and Object")
	}

	groups := make([]responseGroup, 0)

	if err := cursor.All(ctx, &groups); err != nil {
		return derp.Wrap(err, location, "Reading Responses from cursor")
	}

	// Work out which of the competing reactions have been displaced
	displaced := make([]primitive.ObjectID, 0)

	for _, group := range groups {
		displaced = append(displaced, group.displacedResponseIDs()...)
	}

	if len(displaced) == 0 {
		return nil
	}

	// Remove them in a single round-trip.  The DeleteResult is discarded rather than read for
	// its count: it is only ever nil alongside an error, but proving that means trusting a
	// bitmask deep inside the driver, and the log below has the same number already.
	if _, err := collection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": displaced}}); err != nil {
		return derp.Wrap(err, location, "Deleting duplicate Responses")
	}

	log.Info().
		Str("database", collection.Database().Name()).
		Int("count", len(displaced)).
		Msg("Removed duplicate and contradictory Responses")

	// Nothing to see here. Move along.
	return nil
}

// displacedResponseIDs returns the ID of every reaction in this group that must be removed,
// keeping the newest from each set of contradictory types.  A User who Liked *and* Shared one
// Object keeps both; a User who Liked *and* Disliked it keeps only the newer of the two.
func (group responseGroup) displacedResponseIDs() []primitive.ObjectID {

	// Sort the reactions into buckets that displace each other.  Types that contradict share a
	// key, so Like and Dislike land together -- the same rule service.SetResponse applies.
	buckets := make(map[string][]responseRecord)

	for _, record := range group.Responses {
		key := strings.Join(model.ConflictingResponseTypes(record.Type), ",")
		buckets[key] = append(buckets[key], record)
	}

	result := make([]primitive.ObjectID, 0)

	for _, bucket := range buckets {

		survivor := newestResponseID(bucket)

		for _, record := range bucket {
			if record.ResponseID != survivor {
				result = append(result, record.ResponseID)
			}
		}
	}

	return result
}

// newestResponseID returns the ID of the most recently created record in a non-empty bucket.
func newestResponseID(bucket []responseRecord) primitive.ObjectID {

	newest := bucket[0]

	// Ties (reactions created in the same millisecond) keep the first one seen.  Which one
	// survives does not matter; that exactly one survives does.
	for _, record := range bucket[1:] {
		if record.CreateDate > newest.CreateDate {
			newest = record
		}
	}

	return newest.ResponseID
}

// indexExists returns TRUE if an index with the provided name is present on the collection.
func indexExists(ctx context.Context, collection *mongo.Collection, indexName string) (bool, error) {

	const location = "queries.sync.indexExists"

	cursor, err := collection.Indexes().List(ctx)

	if err != nil {
		return false, derp.Wrap(err, location, "Listing indexes", collection.Name())
	}

	indexes := make([]struct {
		Name string `bson:"name"`
	}, 0)

	if err := cursor.All(ctx, &indexes); err != nil {
		return false, derp.Wrap(err, location, "Reading indexes from cursor", collection.Name())
	}

	for _, index := range indexes {
		if index.Name == indexName {
			return true, nil
		}
	}

	return false, nil
}
