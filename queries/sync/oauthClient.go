package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// OAuthClient synchronizes the MongoDB indexes for the OAuthClient collection
func OAuthClient(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "OAuthClient").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("OAuthClient"), indexer.IndexSet{

		// idx_OAuthClient_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_OAuthClient_Recycle": recycleIndex(),
	})
}
