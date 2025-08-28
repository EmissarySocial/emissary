package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Identity(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Identity").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Identity"), indexer.IndexSet{

		"idx_Identity_EmailAddress": mongo.IndexModel{
			Keys: bson.D{
				{Key: "emailAddress", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},

		"idx_Identity_WebfingerUsername": mongo.IndexModel{
			Keys: bson.D{
				{Key: "webfingerUsername", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},

		"idx_Identity_ActivityPubActor": mongo.IndexModel{
			Keys: bson.D{
				{Key: "activityPubActor", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},
	})
}
