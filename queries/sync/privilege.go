package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Privilege(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Privilege").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Privilege"), indexer.IndexSet{

		"idx_Privilege_Identifier": mongo.IndexModel{
			Keys: bson.D{
				{Key: "identifierType", Value: 1},
				{Key: "identifierValue", Value: 1},
			},
		},

		"idx_Privilege_Identity": mongo.IndexModel{
			Keys: bson.D{
				{Key: "identityId", Value: 1},
			},
		},

		"idx_Privilege_Circle": mongo.IndexModel{
			Keys: bson.D{
				{Key: "circleId", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},

		"idx_Privilege_Product": mongo.IndexModel{
			Keys: bson.D{
				{Key: "productId", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},

		"idx_Privilege_RemotePurchase": mongo.IndexModel{
			Keys: bson.D{
				{Key: "remotePurchaseId", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},
	})
}
