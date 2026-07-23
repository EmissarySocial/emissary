package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

func StreamDraft(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "StreamDraft").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("StreamDraft"), indexer.IndexSet{

		// idx_StreamDraft_Recycle serves the nightly RecycleDomain purge (deleteDate > 0).
		"idx_StreamDraft_Recycle": recycleIndex(),
	})
}
