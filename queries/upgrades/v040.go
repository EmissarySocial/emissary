package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version40 rewrites the `syndication` property of every Stream and StreamDraft from the
// legacy document, {values: [...]}, into a plain array. It is safe to run more than once.
func Version40(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version40"

	fmt.Println("... Version 40")

	// Both collections store a model.Stream
	for _, name := range []string{"Stream", "StreamDraft"} {

		modified, err := flattenSyndication(ctx, session.Collection(name))

		if err != nil {
			return derp.Wrap(err, location, "Converting syndication to a plain array", name)
		}

		fmt.Println("...... converted " + fmt.Sprint(modified) + " records in " + name)
	}

	return nil
}

// flattenSyndication replaces each legacy `syndication` document in a collection with the
// array it holds, and returns the number of records it changed
func flattenSyndication(ctx context.Context, collection *mongo.Collection) (int64, error) {

	const location = "queries.upgrades.flattenSyndication"

	// RULE: Only a document is matched. A record that already holds an array, or has no
	// `syndication` at all, is left alone, which is what makes a second run change nothing.
	filter := bson.M{"syndication": bson.M{"$type": "object"}}

	// The server rewrites each record in place. A nil list was stored as a null, so it
	// becomes an empty array.
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "syndication", Value: bson.D{
				{Key: "$ifNull", Value: bson.A{"$syndication.values", bson.A{}}},
			}},
		}}},
	}

	result, err := collection.UpdateMany(ctx, filter, pipeline)

	if err != nil {
		return 0, derp.Wrap(err, location, "Updating records")
	}

	// An unacknowledged write reports no count
	if result == nil {
		return 0, nil
	}

	return result.ModifiedCount, nil
}
