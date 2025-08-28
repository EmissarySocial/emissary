package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Queue synchronizes the Queue collection in the SHARED DATABASE.
func Queue(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Queue").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Queue"), indexer.IndexSet{

		"idx_Queue_NotifiedDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "timeoutDate", Value: 1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{
				"notifiedDate": 0,
			}),
		},

		"idx_Queue_Signature": mongo.IndexModel{
			Keys: bson.D{
				{Key: "signature", Value: 1},
			},
		},
	})
}
