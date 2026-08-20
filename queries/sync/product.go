package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// Product synchronizes the MongoDB indexes for the Product collection
func Product(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Product").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Product"), indexer.IndexSet{

		// idx_Product_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Product_Recycle": recycleIndex(),
	})
}
