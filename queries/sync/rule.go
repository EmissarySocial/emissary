package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Rule synchronizes the MongoDB indexes for the Rule collection
func Rule(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Rule").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Rule"), indexer.IndexSet{

		// idx_Rule_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Rule_Recycle": recycleIndex(),

		// Serves the admin block queries (QueryDomainBlocks / QueryBlockedActors) on the canonical
		// Type/Trigger fields. (idx_Rule_User_Public was dropped: post-D9 nothing is public, so it
		// indexed zero documents.)
		"idx_Rule_User": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "type", Value: 1},
				{Key: "trigger", Value: 1},
				{Key: "followingId", Value: 1},
			},
		},

		// The disposition engine's query (userId IN [me, nil] AND matchKey IN [...]) and the UNIQUE
		// dedup invariant. Unique with no partial filter: hard delete (D16) leaves no tombstone to
		// collide with, so the database enforces one rule per (userId, matchKey). The legacy rows
		// that predate this key are reconciled once by upgrades.Version27, which runs BEFORE this
		// sync so the index can build.
		"idx_Rule_MatchKey": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "matchKey", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	})
}
