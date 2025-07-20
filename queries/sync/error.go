package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Error synchronizes the Error collection in the SHARED DATABASE.
func Error(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Error").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Log"), indexer.IndexSet{

		"idx_Error_CreateDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "createDate", Value: 1},
			},
		},
	})
}
