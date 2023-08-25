package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func ForEachRecord(collection *mongo.Collection, fn func(value mapof.Any) error) error {

	ctx := context.Background()

	cursor, err := collection.Find(ctx, bson.M{})

	if err != nil {
		return derp.Wrap(err, "queries.upgrades.EachRecord", "Error listing records")
	}

	for cursor.Next(ctx) {
		value := mapof.NewAny()

		// Try to read the next record from the cursor
		err := cursor.Decode(&value)

		if err != nil {
			derp.Report(derp.Wrap(err, "queries.upgrades.EachRecord", "Error decoding record"))
			continue
		}

		// Try to map the value into something new
		err = fn(value)

		if err != nil {
			derp.Report(derp.Wrap(err, "queries.upgrades.EachRecord", "Error processing record", value))
			continue
		}

		// Try to update the record back into the database
		_, err = collection.ReplaceOne(ctx, bson.M{"_id": value["_id"]}, value)

		if err != nil {
			derp.Report(derp.Wrap(err, "queries.upgrades.EachRecord", "Error saving record", value))
			continue
		}

		fmt.Print(".")
	}

	return nil
}
