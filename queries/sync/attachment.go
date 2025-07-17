package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Attachment(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Attachment").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Attachment"), indexer.IndexSet{

		"idx_Attachment_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "objectType", Value: 1},
				{Key: "objectId", Value: 1},
				{Key: "category", Value: 1},
			},
		},
	})
}
