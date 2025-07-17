package sync

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

func SearchQuery(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "SearchQuery").Msg("COLLECTION:")

	return nil

	/*
		return indexer.Sync(ctx, database.Collection("SearchQuery"), indexer.IndexSet{

			"idx_SearchQuery_Parent": mongo.IndexModel{
				Keys: bson.D{
					{Key: "parentType", Value: 1},
					{Key: "parentId", Value: 1},
				},
			},
		})
	*/
}
