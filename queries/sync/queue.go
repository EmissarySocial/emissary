package sync

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// Queue synchronizes the Queue collection in the SHARED DATABASE.
func Queue(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Queue").Msg("COLLECTION:")

	return nil
	/*
		return indexer.Sync(ctx, database.Collection("Queue"), indexer.IndexSet{

			"idx_Queue_Identifier": mongo.IndexModel{
				Keys: bson.D{
					{Key: "identifierType", Value: 1},
					{Key: "identifierValue", Value: 1},
				},
			},
		})
	*/
}
