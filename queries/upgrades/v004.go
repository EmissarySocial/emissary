package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version4 migrates Stream.Document.* fields => Stream.* fields
func Version4(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version4"

	streamCollection := session.Collection("Stream")

	fmt.Println("... Version 4")

	cursor, err := streamCollection.Find(ctx, map[string]any{})

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving streams iterator")
	}

	for record := mapof.NewAny(); cursor.Next(ctx); record = mapof.NewAny() {

		if err := cursor.Decode(&record); err != nil {
			return derp.Wrap(err, location, "Unable to decode stream record")
		}

		document := record.GetMap("document")
		for key, value := range document {
			record[key] = value
		}
		delete(record, "document")

		filter := bson.M{"_id": record["_id"]}

		if _, err := streamCollection.ReplaceOne(ctx, filter, record); err != nil {
			return derp.Wrap(err, location, "Unable to update stream record")
		}

		fmt.Print(".")
	}

	return nil
}
