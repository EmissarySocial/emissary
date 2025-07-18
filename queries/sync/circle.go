package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Circle(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Circle").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Circle"), indexer.IndexSet{

		"idx_Circle_User_Name": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "name", Value: 1},
			},
		},

		"idx_Circle_User_Product": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "productId", Value: 1},
			},
		},
	})
}
