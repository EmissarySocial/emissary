package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Stream(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Stream").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Stream"), indexer.IndexSet{

		"idx_Stream_Parent": mongo.IndexModel{
			Keys: bson.D{
				{Key: "parentId", Value: 1},
				{Key: "rank", Value: 1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},

		"idx_Stream_Token": mongo.IndexModel{
			Keys: bson.D{
				{Key: "token", Value: 1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},

		"idx_Stream_PublishDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "publishDate", Value: -1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},

		"idx_Stream_UnPublishDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "unpublishDate", Value: 1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},

		"idx_Stream_Privileges": mongo.IndexModel{
			Keys: bson.D{
				{Key: "privilegeIds", Value: 1},
			},
			Options: options.Index().
				SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},
	})
}
