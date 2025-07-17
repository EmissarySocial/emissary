package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Outbox(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Outbox").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Outbox"), indexer.IndexSet{

		"idx_Outbox_Parent": mongo.IndexModel{
			Keys: bson.D{
				{Key: "parentType", Value: 1},
				{Key: "parentId", Value: 1},
			},
		},
	})
}
