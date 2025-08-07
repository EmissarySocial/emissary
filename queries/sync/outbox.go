package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Outbox(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Outbox").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Outbox"), indexer.IndexSet{

		"idx_Outbox_Actor": mongo.IndexModel{
			Keys: bson.D{
				{Key: "actorType", Value: 1},
				{Key: "actorId", Value: 1},
				{Key: "createDate", Value: -1},
			},
		},

		"idx_Outbox_Permissions": mongo.IndexModel{
			Keys: bson.D{
				{Key: "permissions", Value: 1},
			},
		},

		"idx_Outbox_Object": mongo.IndexModel{
			Keys: bson.D{
				{Key: "objectId", Value: 1},
			},
		},
	})
}
