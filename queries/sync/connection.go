package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Connection(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Connection").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Connection"), indexer.IndexSet{

		"idx_Connection_Type_Active": mongo.IndexModel{
			Keys: bson.D{
				{Key: "type", Value: 1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{
				"active": true,
			}),
		},

		"idx_Connection_Provider": mongo.IndexModel{
			Keys: bson.D{
				{Key: "providerId", Value: 1},
			},
		},
	})

}
