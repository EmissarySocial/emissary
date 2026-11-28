package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// shuffleRange is 2^53, the number of distinct values that MongoDB's $rand can produce
const shuffleRange int64 = 1 << 53

// Shuffle gives every record in the Collection a new random "shuffle" value, in one statement
func Shuffle(ctx context.Context, collection data.Collection) error {

	const location = "queries.Shuffle"

	c := mongoCollection(collection)

	if c == nil {
		return derp.Internal(location, "Collection must be a MongoDB collection")
	}

	// The database picks each value, so no record travels to this server and back
	pipeline := mongo.Pipeline{
		{{Key: "$set", Value: bson.M{
			"shuffle": bson.M{"$toLong": bson.M{"$multiply": bson.A{bson.M{"$rand": bson.M{}}, shuffleRange}}},
		}}},
	}

	if _, err := c.UpdateMany(ctx, bson.M{}, pipeline); err != nil {
		return derp.Wrap(err, location, "Assigning random numbers", c.Name())
	}

	// Every day I'm shufflin'.
	return nil
}
