package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MerchantAccount synchronizes the MongoDB indexes for the MerchantAccount collection
func MerchantAccount(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "MerchantAccount").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("MerchantAccount"), indexer.IndexSet{

		// idx_MerchantAccount_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_MerchantAccount_Recycle": recycleIndex(),

		"idx_MerchantAccount_User": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "_id", Value: 1},
			},
		},
	})

}
