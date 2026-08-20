package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// EncryptionKey synchronizes the MongoDB indexes for the EncryptionKey collection
func EncryptionKey(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "EncryptionKey").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("EncryptionKey"), indexer.IndexSet{

		// idx_EncryptionKey_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_EncryptionKey_Recycle": recycleIndex(),

		"idx_EncryptionKey_Parent": mongo.IndexModel{
			Keys: bson.D{
				{Key: "parentId", Value: 1},
			},
		},
	})
}
