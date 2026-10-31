package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Outbox2 synchronizes the MongoDB indexes for the Outbox2 collection
func Outbox2(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Outbox2").Msg("COLLECTION:")

	// Outbox2 has no recycle index, because service.Outbox2 only hard-deletes (see AGENTS.md)
	return indexer.Sync(ctx, database.Collection("Outbox2"), indexer.IndexSet{

		// Serves an actor's outbox collection, newest first
		"idx_Outbox2_Actor": mongo.IndexModel{
			Keys: bson.D{
				{Key: "actorType", Value: 1},
				{Key: "actorId", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},

		// Serves "does this outbox already hold a Create for this object?"
		"idx_Outbox2_Actor_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "actorType", Value: 1},
				{Key: "actorId", Value: 1},
				{Key: "objectId", Value: 1},
				{Key: "activityType", Value: 1},
			},
		},

		// RULE: One row per activity URL, so a replayed or racing publish cannot store an activity twice.
		// Rows with an empty or missing url are left out, because ActivityURL returns "" for an
		// unknown actor type, and those rows must not collide with one another.
		"idx_Outbox2_URL": mongo.IndexModel{
			Keys: bson.D{
				{Key: "url", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{
				"url": bson.M{"$gt": ""},
			}),
		},

		// Serves lookups by object across every outbox
		"idx_Outbox2_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "objectId", Value: 1},
			},
		},

		// Serves the read filter on permissions
		"idx_Outbox2_Permissions": mongo.IndexModel{
			Keys: bson.D{
				{Key: "permissions", Value: 1},
			},
		},
	})
}
