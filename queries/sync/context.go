package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Collection(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Collection").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Collection"), indexer.IndexSet{

		"idx_Collection_Context": mongo.IndexModel{
			Keys: bson.D{
				{Key: "context", Value: 1},
			},
		},

		"idx_Collection_InReplyTo": mongo.IndexModel{
			Keys: bson.D{
				{Key: "inReplyTo", Value: 1},
			},
		},
	})
}
