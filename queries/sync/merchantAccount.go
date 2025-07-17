package sync

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

func MerchantAccount(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "MerchantAccount").Msg("COLLECTION:")

	return nil
}
