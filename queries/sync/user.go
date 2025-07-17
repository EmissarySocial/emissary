package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func User(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "User").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("User"), indexer.IndexSet{

		"idx_User_Username": mongo.IndexModel{
			Keys: bson.D{
				{Key: "username", Value: 1},
				{Key: "deleteDate", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},

		"idx_User_Email": mongo.IndexModel{
			Keys: bson.D{
				{Key: "emailAddress", Value: 1},
				{Key: "deleteDate", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	})
}
