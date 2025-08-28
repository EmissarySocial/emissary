package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Annotation(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Annotation").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Annotation"), indexer.IndexSet{

		"idx_Annotation_URL": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "url", Value: 1},
			},
		},
	})
}
