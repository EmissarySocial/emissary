package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ForEachFunc func(value mapof.Any) bool

func ForEachRecord(collection *mongo.Collection, fn ForEachFunc) error {

	const location = "queries.upgrades.ForEachRecord"

	if collection == nil {
		panic("collection is nil")
	}

	ctx := context.Background()

	cursor, err := collection.Find(ctx, bson.M{})

	if err != nil {
		return derp.Wrap(err, location, "Error listing records")
	}

	for cursor.Next(ctx) {
		value := mapof.NewAny()

		// Try to read the next record from the cursor
		if err := cursor.Decode(&value); err != nil {
			derp.Report(derp.Wrap(err, location, "Error decoding record"))
			continue
		}

		// Try to map the value into something new
		if changed := fn(value); !changed {
			continue
		}

		// If the record has been changed, then update the database
		if _, err = collection.ReplaceOne(ctx, bson.M{"_id": value["_id"]}, value); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to save record", value))
			continue
		}

		fmt.Print(".")
	}

	return nil
}
