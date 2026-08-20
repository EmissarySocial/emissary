package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// Domain synchronizes the MongoDB indexes for the Domain collection
func Domain(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Domain").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Domain"), indexer.IndexSet{

		// idx_Domain_Recycle serves the nightly RecycleDomain purge (deleteDate > 0). The Domain
		// collection is effectively a singleton, so this index stays empty in practice -- but it
		// keeps the invariant that every recycled collection has a recycle index, so the purge
		// never falls back to a COLLSCAN.
		"idx_Domain_Recycle": recycleIndex(),
	})
}
