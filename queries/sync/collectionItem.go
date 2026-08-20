package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollectionItem synchronizes the MongoDB indexes for the CollectionItem collection
func CollectionItem(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "CollectionItem").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("CollectionItem"), indexer.IndexSet{

		// Enforces one CollectionItem per (collection, uri). This makes
		// SaveUnique concurrency-safe: it inserts optimistically and relies on
		// this index to reject a racing duplicate insert of the same URI.
		//
		// The partial filter scopes uniqueness to live rows (deleteDate == 0),
		// mirroring service.notDeleted(), so a soft-deleted item does not block
		// re-adding the same URI later.
		"idx_CollectionItem_Collection_URI": mongo.IndexModel{
			Keys: bson.D{
				{Key: "collectionId", Value: 1},
				{Key: "uri", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.D{
				{Key: "deleteDate", Value: 0},
			}),
		},

		// Serves the reply list reader: items looked up by inReplyTo, ordered by
		// createDate (QueryByInReplyTo / ReplyLinksAfter).
		"idx_CollectionItem_InReplyTo": mongo.IndexModel{
			Keys: bson.D{
				{Key: "inReplyTo", Value: 1},
				{Key: "createDate", Value: 1},
			},
		},
	})
}
