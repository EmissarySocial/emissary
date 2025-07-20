package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SearchQuery(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "SearchQuery").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("SearchQuery"), indexer.IndexSet{

		"idx_SearchQuery_Signature": mongo.IndexModel{
			Keys: bson.D{
				{Key: "signature", Value: 1},
			},
		},

		"idx_SearchQuery_Parent": mongo.IndexModel{
			Keys: bson.D{
				{Key: "parentType", Value: 1},
				{Key: "parentId", Value: 1},
			},
		},

		"idx_SearchQuery_Notified": mongo.IndexModel{
			Keys: bson.D{
				{Key: "timeoutDate", Value: 1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"notifiedDate": 0}),
		},
	})
}
