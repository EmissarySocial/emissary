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

		"idx_DigitalDome_Lock": mongo.IndexModel{
			Keys: bson.D{
				{Key: "lockId", Value: 1},
				{Key: "startDate", Value: 1},
				{Key: "priority", Value: 1},
			},
		},

		"idx_DigitalDome_StartDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "startDate", Value: -1},
				{Key: "timeoutDate", Value: -1},
			},
		},

		"idx_DigitalDome_Signature": mongo.IndexModel{
			Keys: bson.D{
				{Key: "signature", Value: 1},
			},
		},
	})
}
