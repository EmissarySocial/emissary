package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Log synchronizes the Log collection in the SHARED DATABASE.
func Log(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Log").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Log"), indexer.IndexSet{

		"idx_Log_CreateDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "createDate", Value: -1},
			},
		},
	})
}
