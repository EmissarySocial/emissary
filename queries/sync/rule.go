package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Rule(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Rule").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Rule"), indexer.IndexSet{

		"idx_Rule_User": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "type", Value: 1},
				{Key: "trigger", Value: 1},
				{Key: "followingId", Value: 1},
			},
		},

		"idx_Rule_User_Public": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "type", Value: 1},
				{Key: "trigger", Value: 1},
				{Key: "publishDate", Value: -1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.D{
				{Key: "public", Value: true},
			}),
		},
	})
}
