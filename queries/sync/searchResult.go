package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func SearchResult(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "SearchResult").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("SearchResult"), indexer.IndexSet{

		"idx_SearchResult_Date": mongo.IndexModel{
			Keys: bson.D{
				{Key: "date", Value: 1},
			},
		},

		"idex_SearchResult_ReindexDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "reindexDate", Value: 1},
			},
		},

		"idx_SearchResult_Type_Tags": mongo.IndexModel{
			Keys: bson.D{
				{Key: "type", Value: 1},
				{Key: "tags", Value: 1},
			},
		},

		"idx_SearchResult_Type_Index": mongo.IndexModel{
			Keys: bson.D{
				{Key: "type", Value: 1},
				{Key: "index", Value: 1},
			},
		},

		"idx_SearchResult_URL": mongo.IndexModel{
			Keys: bson.D{
				{Key: "url", Value: 1},
			},
		},
	})
}
