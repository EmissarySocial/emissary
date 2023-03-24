package queries

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpgradeMongoDB(connectionString string, databaseName string, domain *model.Domain) error {

	const currentDatabaseVersion = 1
	const location = "queries.UpgradeMongoDB"

	// If we're already at the target database version, then skip any other work
	if domain.DatabaseVersion == currentDatabaseVersion {
		return nil
	}

	ctx := context.Background()
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionString))

	if err != nil {
		return derp.Wrap(err, "data.mongodb.New", "Error creating mongodb client")
	}

	if err := client.Connect(ctx); err != nil {
		return derp.Wrap(err, "data.mongodb.New", "Error connecting to mongodb Server")
	}

	session := client.Database(databaseName)

	fmt.Println("============ UPGRADING DATABASE ============")

	// Upgrade from version 0 to version 1
	if domain.DatabaseVersion < 1 {
		streamCollection := session.Collection("Stream")

		cursor, err := streamCollection.Find(ctx, map[string]any{})

		if err != nil {
			return derp.Wrap(err, location, "Error retrieving streams iterator")
		}

		record := mapof.NewAny()

		for cursor.Next(ctx) {

			if err := cursor.Decode(&record); err != nil {
				return derp.Wrap(err, location, "Error decoding stream record")
			}

			delete(record, "inReplyTo")

			switch record.GetString("templateId") {
			case "outbox-item":
				record["socialRole"] = "Note"
			case "folder":
				record["socialRole"] = "Page"
			case "photograph":
				record["socialRole"] = "Image"
			default:
				record["socialRole"] = "Article"
			}

			filter := bson.M{"_id": record["_id"]}

			if _, err := streamCollection.ReplaceOne(ctx, filter, record); err != nil {
				return derp.Wrap(err, location, "Error updating stream record")
			}

			fmt.Print(".")
			record = mapof.NewAny()
		}
	}

	domainCollection := session.Collection("Domain")

	filter := bson.M{"_id": primitive.NilObjectID}
	update := bson.M{"$set": bson.M{"databaseVersion": currentDatabaseVersion}}

	if _, err := domainCollection.UpdateOne(ctx, filter, update); err != nil {
		return derp.Wrap(err, location, "Error updating domain record")
	}

	fmt.Println(".")
	fmt.Println("DONE.")
	return nil
}
