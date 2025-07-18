package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// DigitalDome synchronizes the DigitalDome collection in the SHARED DATABASE.
func DigitalDome(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "DigitalDome").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("DigitalDome"), indexer.IndexSet{

		"idx_DigitalDome_CreateDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "createDate", Value: -1},
			},
		},
	})
}
