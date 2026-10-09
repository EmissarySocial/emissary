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

		// The weekly purge in consumer.PurgeErrors reads this one
		"idx_ErrorLog_CreateDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "createDate", Value: 1},
			},
		},

		// Every occurrence of one defect shares a signature, newest first
		"idx_ErrorLog_Signature": mongo.IndexModel{
			Keys: bson.D{
				{Key: "signature", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},

		// The triage queue reads the errors that have not been decided yet
		"idx_ErrorLog_Status": mongo.IndexModel{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},
	})
}
