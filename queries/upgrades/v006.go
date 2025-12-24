package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version6 moves outbox-message summaries => outbox-message content
func Version6(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version6"

	streamCollection := session.Collection("Stream")

	fmt.Println("... Version 6")

	cursor, err := streamCollection.Find(ctx, map[string]any{})

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving streams iterator")
	}

	for record := mapof.NewAny(); cursor.Next(ctx); record = mapof.NewAny() {

		if err := cursor.Decode(&record); err != nil {
			return derp.Wrap(err, location, "Error decoding stream record")
		}

		if inReplyTo, ok := record["inReplyTo"]; ok {

			if mappedValue, ok := inReplyTo.(mapof.Any); ok {
				record["inReplyTo"] = mappedValue.GetString("url")

				filter := bson.M{"_id": record["_id"]}

				if _, err := streamCollection.ReplaceOne(ctx, filter, record); err != nil {
					return derp.Wrap(err, location, "Error updating stream record")
				}

				fmt.Print(".")
			}
		}
	}

	return nil
}
