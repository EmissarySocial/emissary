package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version8 moves all `journal.*` fields into the top level of each model object
func Version8(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version8"

	collections := []string{"Attachment", "Block", "Domain", "EncryptionKey", "Folder", "Follower", "Following", "Group", "Inbox", "Mention", "Outbox", "Response", "Stream", "StreamDraft", "User"}

	fmt.Println("... Version 8")

	for _, collectionName := range collections {
		collection := session.Collection(collectionName)

		cursor, err := collection.Find(ctx, map[string]any{})

		if err != nil {
			return derp.Wrap(err, location, "Error retrieving streams iterator")
		}

		record := mapof.NewAny()

		for cursor.Next(ctx) {

			// Read the record from the database
			if err := cursor.Decode(&record); err != nil {
				return derp.Wrap(err, location, "Error decoding stream record")
			}

			// Update the record
			journal := record.GetMap("journal")

			for key, value := range journal {
				record[key] = value
			}

			delete(record, "journal")

			// Replace the record back to the database
			filter := bson.M{"_id": record["_id"]}

			if _, err := collection.ReplaceOne(ctx, filter, record); err != nil {
				return derp.Wrap(err, location, "Error updating stream record")
			}

			fmt.Print(".")
			record = mapof.NewAny()
		}
	}

	return nil
}
