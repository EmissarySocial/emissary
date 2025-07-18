package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Mention(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Mention").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Mention"), indexer.IndexSet{

		"idx_Mention_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "objectId", Value: 1},
				{Key: "type", Value: 1},
			},
		},
	})
}
