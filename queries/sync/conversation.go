package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Conversation(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Conversation").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Conversation"), indexer.IndexSet{

		"idx_Conversation_UpdateDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "updateDate", Value: -1},
			},
		},
	})
}
