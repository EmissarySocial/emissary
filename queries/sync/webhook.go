package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Webhook(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Webhook").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Webhook"), indexer.IndexSet{

		"idx_Webhook_Event": mongo.IndexModel{
			Keys: bson.D{
				{Key: "events", Value: 1},
			},
		},
	})
}
