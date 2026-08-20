package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// JWT synchronizes the MongoDB indexes for the JWT collection
func JWT(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "JWT").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("JWT"), indexer.IndexSet{

		// idx_JWT_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_JWT_Recycle": recycleIndex(),

		"idx_JWT_Key": mongo.IndexModel{
			Keys: bson.D{
				{Key: "keyName", Value: 1},
			},
		},
	})
}
