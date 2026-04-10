package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Context(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Context").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Context"), indexer.IndexSet{

		"idx_Context_Context": mongo.IndexModel{
			Keys: bson.D{
				{Key: "context", Value: 1},
			},
		},

		"idx_Context_InReplyTo": mongo.IndexModel{
			Keys: bson.D{
				{Key: "inReplyTo", Value: 1},
			},
		},
	})
}
