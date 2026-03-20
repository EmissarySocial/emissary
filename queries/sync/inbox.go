package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Inbox(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Inbox").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Inbox"), indexer.IndexSet{

		"idx_Inbox_ActivityID": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "activityId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},

		"idx_Inbox_DirectMessages": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{"isPublic": false}),
		},
	})
}
