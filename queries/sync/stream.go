package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Stream(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Stream").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Stream"), indexer.IndexSet{

		"idx_Stream_Parent_Rank": mongo.IndexModel{
			Keys: bson.D{
				{Key: "parentId", Value: 1},
				{Key: "rank", Value: 1},
			},
		},

		"idx_Stream_Token": mongo.IndexModel{
			Keys: bson.D{
				{Key: "token", Value: 1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},
	})
}
