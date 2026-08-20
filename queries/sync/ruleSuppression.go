package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RuleSuppression synchronizes the MongoDB indexes for the RuleSuppression collection
func RuleSuppression(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "RuleSuppression").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("RuleSuppression"), indexer.IndexSet{

		// Serves the backfill's IsSuppressed lookup (P7-3), and enforces one suppression per
		// (owner, remote entry). UNIQUE: Suppress is idempotent-by-read, and the index backstops
		// the race two concurrent deletes could otherwise win together.
		"idx_RuleSuppression_Remote": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "remoteId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},

		// Serves DeleteByFollowingID when a provider subscription is removed
		"idx_RuleSuppression_Following": mongo.IndexModel{
			Keys: bson.D{
				{Key: "followingId", Value: 1},
			},
		},
	})
}
