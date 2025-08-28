package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Response(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Response").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Response"), indexer.IndexSet{

		"idx_Response_User": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "type", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},

		"idx_Response_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "object", Value: 1},
				{Key: "type", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},
	})
}
