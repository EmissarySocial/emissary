package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version7 moves outbox-message summaries => outbox-message content
func Version7(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version7"

	streamCollection := session.Collection("Stream")

	fmt.Println("... Version 7")

	cursor, err := streamCollection.Find(ctx, map[string]any{"templateId": "outbox-message"})

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving streams iterator")
	}

	record := mapof.NewAny()

	for cursor.Next(ctx) {

		if err := cursor.Decode(&record); err != nil {
			return derp.Wrap(err, location, "Error decoding stream record")
		}

		if summary := record.GetString("summary"); summary != "" {

			delete(record, "summary")
			record["content"] = map[string]any{
				"html": summary,
				"raw":  summary,
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
