package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Attachment synchronizes the MongoDB indexes for the Attachment collection
func Attachment(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Attachment").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Attachment"), indexer.IndexSet{

		// idx_Attachment_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Attachment_Recycle": recycleIndex(),

		"idx_Attachment_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "objectType", Value: 1},
				{Key: "objectId", Value: 1},
				{Key: "category", Value: 1},
				{Key: "rank", Value: 1},
			},
		},
	})
}
