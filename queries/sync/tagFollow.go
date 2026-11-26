package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TagFollow synchronizes the MongoDB indexes for the TagFollow collection
func TagFollow(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "TagFollow").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("TagFollow"), indexer.IndexSet{

		// idx_TagFollow_User_Value lists a User's followed tags, and lets them follow each tag only once
		"idx_TagFollow_User_Value": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "value", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},

		// idx_TagFollow_Value finds everyone who follows a post's tags
		"idx_TagFollow_Value": mongo.IndexModel{
			Keys:    bson.D{{Key: "value", Value: 1}},
			Options: options.Index().SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},
	})
}
