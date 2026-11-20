package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LinkPreview synchronizes the LinkPreview collection in the SHARED DATABASE.
func LinkPreview(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "LinkPreview").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("LinkPreview"), indexer.IndexSet{

		// MongoDB deletes each record once its expiresAt passes, so nothing else purges the cache
		"idx_LinkPreview_Expires": mongo.IndexModel{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	})
}
