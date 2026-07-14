package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func PushSubscription(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "PushSubscription").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("PushSubscription"), indexer.IndexSet{

		"idx_PushSubscription_User": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},

		"idx_PushSubscription_Endpoint": mongo.IndexModel{
			Keys: bson.D{
				{Key: "endpoint", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	})
}
