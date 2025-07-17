package sync

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

func Privilege(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Privilege").Msg("COLLECTION:")

	return nil

	/*
		return indexer.Sync(ctx, database.Collection("Privilege"), indexer.IndexSet{

			"idx_Privilege_Parent": mongo.IndexModel{
				Keys: bson.D{
					{Key: "parentType", Value: 1},
					{Key: "parentId", Value: 1},
				},
			},
		})
	*/
}
