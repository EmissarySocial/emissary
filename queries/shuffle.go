package queries

import (
	"context"
	"math/rand/v2"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Shuffle randomizes the "shuffle" field for each record in the Collection
func Shuffle(ctx context.Context, collection data.Collection) error {

	const location = "queries.Shuffle"

	c := mongoCollection(collection)

	if c == nil {
		return derp.InternalError(location, "Collection must be a MongoDB collection")
	}

	if err := shuffleA(ctx, c); err != nil {
		return derp.Wrap(err, location, "Error assigning random numbers", nil)
	}

	if err := shuffleB(ctx, c); err != nil {
		return derp.Wrap(err, location, "Error assigning random numbers", nil)
	}

	// Oh yeah.
	return nil
}

// shuffleA assigns a random number to the "shuffle" field for each record in the Collection
func shuffleA(ctx context.Context, collection *mongo.Collection) error {

	const location = "queries.Shuffle"

	opts := options.FindOptions{
		Projection: bson.M{"_id": 1},
	}

	// Scan through all records (unsorted)
	cursor, err := collection.Find(ctx, bson.M{}, &opts)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving records", pipeline)
	}

	// For each record in the Collection
	for cursor.Next(ctx) {

		// Decode the result (just the ID)
		result := mapof.NewAny()
		if err := cursor.Decode(&result); err != nil {
			return derp.Wrap(err, location, "Error decoding record", pipeline)
		}

		// Set a new random value for the "shuffle" field
		_, err := collection.UpdateOne(
			ctx,
			bson.M{"_id": result["_id"]},
			bson.M{"$set": bson.M{"shuffle": rand.Int64()}},
		)

		if err != nil {
			return derp.Wrap(err, location, "Error updating record", result)
		}
	}

	// Oh yeah.
	return nil
}

// shuffleB sets the "shuffle" field to a sequential number
func shuffleB(ctx context.Context, collection *mongo.Collection) error {

	const location = "queries.Shuffle"

	var shuffle int64

	opts := options.FindOptions{
		Projection: bson.M{"_id": 1},
		Sort:       bson.M{"shuffle": 1},
	}

	// Scan through all records (sorted by "shuffle")
	cursor, err := collection.Find(ctx, bson.M{}, &opts)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving records", pipeline)
	}

	// For each record in the Collection
	for cursor.Next(ctx) {

		// Decode the result
		result := mapof.NewAny()
		if err := cursor.Decode(&result); err != nil {
			return derp.Wrap(err, location, "Error decoding record", pipeline)
		}

		// Set a sequential value for the "shuffle" field
		shuffle++
		_, err := collection.UpdateOne(
			ctx,
			bson.M{"_id": result["_id"]},
			bson.M{"$set": bson.M{"shuffle": shuffle}},
		)

		if err != nil {
			return derp.Wrap(err, location, "Error updating record", result)
		}
	}

	// Oh yeah.
	return nil
}
