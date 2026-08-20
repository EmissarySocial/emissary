package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Inbox synchronizes the MongoDB indexes for the Inbox collection
func Inbox(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "Inbox").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("Inbox"), indexer.IndexSet{

		"idx_Inbox_ActivityID": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "activityId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},

		"idx_Inbox_DirectMessages": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().SetPartialFilterExpression(bson.M{"isPublic": false}),
		},

		// idx_Inbox_User serves the general inbox listing (Inbox.RangeByUser), which filters by
		// userId and sorts by _id. The DirectMessages index above is partial on isPublic:false, so
		// it cannot serve an all-messages listing; without this non-partial index MongoDB blocking-
		// sorts the user's whole inbox on every page.
		"idx_Inbox_User": mongo.IndexModel{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "_id", Value: 1},
			},
		},
	})
}
