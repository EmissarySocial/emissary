package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func SearchTag(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "SearchTag").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("SearchTag"), indexer.IndexSet{

		"idx_SearchTag_Value": mongo.IndexModel{
			Keys: bson.D{
				{Key: "value", Value: 1},
				{Key: "stateId", Value: 1},
			},
		},

		"idx_SearchTag_State_Name": mongo.IndexModel{
			Keys: bson.D{
				{Key: "stateId", Value: 1},
				{Key: "name", Value: 1},
			},
		},

		"idx_SearchTag_IsFeatured_Rank": mongo.IndexModel{
			Keys: bson.D{
				{Key: "isFeatured", Value: 1},
				{Key: "rank", Value: 1},
			},
		},
	})
}
