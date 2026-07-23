package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Notification(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Notification").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Notification"), indexer.IndexSet{

		// idx_Notification_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Notification_Recycle": recycleIndex(),

		"idx_Notification_User": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "createDate", Value: -1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},

		"idx_Notification_Unread": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "readDate", Value: 1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},

		"idx_Notification_Activity": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "activityId", Value: 1},
			},
		},

		// Supports the daily retention purge (Notification.PurgeBefore), which spans every User
		// and so cannot use any of the userId-prefixed indexes above.  No partial filter: the
		// purge hard-deletes soft-deleted rows too.
		"idx_Notification_Purge": mongo.IndexModel{
			Keys: bson.D{
				{Key: "createDate", Value: 1},
			},
		},

		"idx_Notification_Stream": mongo.IndexModel{
			Keys: bson.D{
				{Key: "streamId", Value: 1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},
	})
}
