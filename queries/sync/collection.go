package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Collection(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Collection").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Collection"), indexer.IndexSet{

		// idx_Collection_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_Collection_Recycle": recycleIndex(),

		// Enforces one Collection per (parentId, collectionType). This is what makes JIT
		// creation concurrency-safe: LoadOrCreateByParent inserts optimistically
		// and relies on this index to reject the loser of a create race.
		//
		// The partial filter scopes uniqueness to live rows (deleteDate == 0),
		// mirroring service.notDeleted(), so that a soft-deleted collection does
		// not block re-creation of a fresh one with the same (parentId, collectionType).
		"idx_Collection_Parent_Type": mongo.IndexModel{
			Keys: bson.D{
				{Key: "parentId", Value: 1},
				{Key: "collectionType", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.D{
				{Key: "deleteDate", Value: 0},
			}),
		},
	})
}
