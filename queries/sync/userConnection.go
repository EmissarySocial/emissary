package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserConnection synchronizes the MongoDB indexes for the UserConnection collection
func UserConnection(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "UserConnection").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("UserConnection"), indexer.IndexSet{

		// idx_UserConnection_Recycle serves the nightly RecycleDomain purge. These records are
		// HARD deleted, so it should never match anything -- it is here because a soft-deleted
		// row that nothing purges would keep a live credential readable forever.
		"idx_UserConnection_Recycle": recycleIndex(),

		// idx_UserConnection_User_Type enforces one connection per service per User, which is
		// a real unique index only because these records are HARD deleted -- a soft-deleted
		// tombstone would hold the slot and make reconnecting impossible.
		"idx_UserConnection_User_Type": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "type", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	})
}
