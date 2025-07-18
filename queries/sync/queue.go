package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Queue synchronizes the Queue collection in the SHARED DATABASE.
func Queue(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Queue").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Queue"), indexer.IndexSet{

		"idx_Queue_StartDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "startDate", Value: 1},
				{Key: "priority", Value: 1},
			},
		},

		"idx_Queue_Signature": mongo.IndexModel{
			Keys: bson.D{
				{Key: "signature", Value: 1},
			},
		},
	})
}
