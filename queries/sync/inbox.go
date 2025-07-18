package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Inbox(ctx context.Context, database *mongo.Database) error {

	log.Debug().Str("database", database.Name()).Str("collection", "Inbox").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Inbox"), indexer.IndexSet{

		"idx_Inbox_User_Folder": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "folderId", Value: 1},
				{Key: "readDate", Value: 1},
			},
		},

		"idx_Inbox_User_CreateDate": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "createDate", Value: 1},
			},
		},

		"idx_Inbox_User_Following": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "origin.followingId", Value: 1},
			},
		},
	})
}
