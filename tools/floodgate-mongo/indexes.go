package floodgatemongo

import (
	"context"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SyncIndexes creates the indexes that the Store's two list queries use. Both are partial, so
// they hold only the hosts that are in trouble, which are few.
func SyncIndexes(ctx context.Context, database *mongo.Database) error {

	const location = "floodgatemongo.SyncIndexes"

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "dormantSince", Value: 1}, {Key: "nextProbe", Value: 1}},
			Options: options.Index().
				SetName("idx_Host_Dormant").
				SetPartialFilterExpression(bson.M{"dormantSince": bson.M{"$gt": 0}}),
		},
		{
			Keys: bson.D{{Key: "isPaused", Value: 1}},
			Options: options.Index().
				SetName("idx_Host_Paused").
				SetPartialFilterExpression(bson.M{"isPaused": true}),
		},
	}

	if _, err := database.Collection(CollectionName).Indexes().CreateMany(ctx, indexes); err != nil {
		return derp.Wrap(err, location, "Creating indexes")
	}

	return nil
}
