package sync

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// oauthClientURLIndex names the index that allows one live client per URL client_id.
// Its presence doubles as proof that the collection has already been de-duplicated.
const oauthClientURLIndex = "idx_OAuthClient_ClientURL"

// OAuthClient synchronizes the MongoDB indexes for the OAuthClient collection
func OAuthClient(ctx context.Context, database *mongo.Database) error {

	const location = "queries.sync.OAuthClient"

	log.Trace().Str("database", database.Name()).Str("collection", "OAuthClient").Msg("COLLECTION:")

	collection := database.Collection("OAuthClient")

	// Retire any client that would violate the unique index below.  A failure here is reported
	// but not fatal: only the unique index fails to build (and is retried on the next startup).
	if err := deduplicateOAuthClients(ctx, collection); err != nil {
		derp.Report(derp.Wrap(err, location, "Removing duplicate OAuthClients", database.Name()))
	}

	return indexer.Sync(ctx, collection, indexer.IndexSet{

		// idx_OAuthClient_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_OAuthClient_Recycle": recycleIndex(),

		// idx_OAuthClient_ClientURL allows one live client per URL client_id, whichever server creates it.
		// Registered clients carry an empty clientUrl, which the partial filter leaves out.
		oauthClientURLIndex: mongo.IndexModel{
			Keys: bson.D{
				{Key: "clientUrl", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{
					"clientUrl":  bson.M{"$gt": ""},
					"deleteDate": 0,
				}),
		},
	})
}

// deduplicateOAuthClients soft-deletes every live client that shares its clientUrl with an older one,
// keeping the oldest.  Once the unique index exists, later startups skip the scan.
func deduplicateOAuthClients(ctx context.Context, collection *mongo.Collection) error {

	const location = "queries.sync.deduplicateOAuthClients"

	// The index is only ever created after a successful cleanup, so its presence proves that
	// there is nothing left to find.
	indexed, err := indexExists(ctx, collection, oauthClientURLIndex)

	if err != nil {
		return derp.Wrap(err, location, "Looking for the unique index")
	}

	if indexed {
		return nil
	}

	// Collect every clientUrl named by more than one live client, oldest first
	pipeline := []bson.M{
		{"$match": bson.M{"clientUrl": bson.M{"$gt": ""}, "deleteDate": 0}},
		{"$sort": bson.M{"_id": 1}},
		{"$group": bson.M{
			"_id":       "$clientUrl",
			"clientIds": bson.M{"$push": "$_id"},
		}},
		{"$match": bson.M{"clientIds.1": bson.M{"$exists": true}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)

	if err != nil {
		return derp.Wrap(err, location, "Grouping OAuthClients by clientUrl")
	}

	groups := make([]struct {
		ClientIDs []primitive.ObjectID `bson:"clientIds"`
	}, 0)

	if err := cursor.All(ctx, &groups); err != nil {
		return derp.Wrap(err, location, "Reading OAuthClients from cursor")
	}

	// Every client after the oldest in its group is a duplicate
	duplicates := make([]primitive.ObjectID, 0)

	for _, group := range groups {
		duplicates = append(duplicates, group.ClientIDs[1:]...)
	}

	if len(duplicates) == 0 {
		return nil
	}

	// Soft-delete them, so their grants stay readable until the nightly purge
	update := bson.M{"$set": bson.M{"deleteDate": time.Now().UnixMilli()}}

	if _, err := collection.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": duplicates}}, update); err != nil {
		return derp.Wrap(err, location, "Deleting duplicate OAuthClients")
	}

	log.Info().
		Str("database", collection.Database().Name()).
		Int("count", len(duplicates)).
		Msg("Removed duplicate OAuthClients")

	// Nothing to see here. Move along.
	return nil
}
