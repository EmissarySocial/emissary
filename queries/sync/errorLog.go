package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ErrorLog synchronizes the ErrorLog collection in the SHARED DATABASE.
func ErrorLog(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "ErrorLog").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("ErrorLog"), indexer.IndexSet{

		"idx_ErrorLog_CreateDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "createDate", Value: 1},
			},
		},
	})
}
