package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Group(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Group").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Group"), indexer.IndexSet{

		"idx_Group_Label": mongo.IndexModel{
			Keys: bson.D{
				{Key: "label", Value: 1},
			},
		},
	})
}
