package sync

import (
	"context"

	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TagRelay synchronizes the MongoDB indexes for the TagRelay collection
func TagRelay(ctx context.Context, database *mongo.Database) error {

	log.Trace().Str("database", database.Name()).Str("collection", "TagRelay").Msg("COLLECTION:")

	return indexer.Sync(ctx, database.Collection("TagRelay"), indexer.IndexSet{

		// idx_TagRelay_Value keeps one relay subscription per tag
		"idx_TagRelay_Value": mongo.IndexModel{
			Keys:    bson.D{{Key: "value", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},

		// idx_TagRelay_ActorURL finds the subscription an inbound activity belongs to
		"idx_TagRelay_ActorURL": mongo.IndexModel{
			Keys:    bson.D{{Key: "actorUrl", Value: 1}},
			Options: options.Index().SetPartialFilterExpression(bson.M{"deleteDate": 0}),
		},
	})
}
