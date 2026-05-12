package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// SigninAttempt synchronizes the SigninAttempt collection in the SHARED DATABASE.
func SigninAttempt(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "SigninAttempt").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("SigninAttempt"), indexer.IndexSet{

		"idx_SigninAttempt_Username": mongo.IndexModel{
			Keys: bson.D{
				{Key: "username", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},
	})
}
