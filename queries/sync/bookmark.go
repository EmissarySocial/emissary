package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Bookmark synchronizes the MongoDB indexes for the Bookmark collection
func Bookmark(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Bookmark").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Bookmark"), indexer.IndexSet{

		// idx_Bookmark_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Bookmark_Recycle": recycleIndex(),

		// One live bookmark per User per post; also serves "is this post bookmarked" lookups
		// and the User's bookmark list (newest first).
		"idx_Bookmark_UserURL": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "url", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},
	})
}
