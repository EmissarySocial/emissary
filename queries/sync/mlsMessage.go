package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func MLSMessage(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "MLSMessage").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("MLSMessage"), indexer.IndexSet{

		"idx_MLSMessage_UserID": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "createDate", Value: 1},
			},
		},
	})
}
