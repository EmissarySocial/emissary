package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// StreamSource synchronizes the MongoDB indexes for the StreamSource collection
func StreamSource(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "StreamSource").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("StreamSource"), indexer.IndexSet{

		// idx_StreamSource_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_StreamSource_Recycle": recycleIndex(),

		// Enforces one live StreamSource record per Stream, and serves LoadByStreamID.  The partial
		// filter lets a deleted record sit beside its replacement until the purge removes it.
		"idx_StreamSource_Stream": mongo.IndexModel{
			Keys: bson.D{
				{Key: "streamId", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},
	})
}
