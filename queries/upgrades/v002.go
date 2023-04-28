package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Version2(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version1"
	streamCollection := session.Collection("Stream")

	fmt.Println("... Version 1")

	cursor, err := streamCollection.Find(ctx, map[string]any{})

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving streams iterator")
	}

	record := mapof.NewAny()

	for cursor.Next(ctx) {

		if err := cursor.Decode(&record); err != nil {
			return derp.Wrap(err, location, "Error decoding stream record")
		}

		document := record.GetMap("document")
		if _, ok := document["attributedTo"]; !ok {
			if author, ok := document["author"]; ok {
				document["attributedTo"] = []any{author}
				delete(document, "author")
				delete(record, "author")
				record["document"] = document
			}
		}

		filter := bson.M{"_id": record["_id"]}

		if _, err := streamCollection.ReplaceOne(ctx, filter, record); err != nil {
			return derp.Wrap(err, location, "Error updating stream record")
		}

		fmt.Print(".")
		record = mapof.NewAny()
	}

	return nil
}
