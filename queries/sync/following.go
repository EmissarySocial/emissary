package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Following(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Following").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Following"), indexer.IndexSet{

		"idx_Following_User_Folder": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "folderId", Value: 1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{
				"deleteDate": 0,
			}),
		},

		"idx_Following_User_Profile": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "profileUrl", Value: 1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{
				"deleteDate": 0,
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
