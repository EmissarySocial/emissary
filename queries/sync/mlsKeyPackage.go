package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func MLSKeyPackage(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "MLSKeyPackage").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("MLSKeyPackage"), indexer.IndexSet{

		"idx_MLSKeyPackage_UserID": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
			},
		},
	})
}
