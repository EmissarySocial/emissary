package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version19...
func Version19(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 19")

	ForEachRecord(session.Collection("Outbox"), func(record mapof.Any) error {

		if parentID, exists := record["parentId"]; exists {
			record["actorId"] = parentID
			delete(record, "parentId")
		}

		if parentType, exists := record["parentType"]; exists {
			record["actorType"] = parentType
			delete(record, "parentType")
		}

		if _, exists := record["type"]; exists {
			record["activityType"] = "Create"
			delete(record, "type")
		}

		if url, exists := record["url"]; exists {
			record["objectId"] = url
			delete(record, "url")
		}

		record["permissions"] = model.NewAnonymousPermissions()

		return nil
	})

	// Update all User Outbox records
	{
		cursor, err := session.Collection("User").Find(ctx, mapof.Any{"deleteDate": 0})

		if err != nil {
			return err
		}

		for cursor.Next(ctx) {
			user := model.NewUser()
			if err := cursor.Decode(&user); err != nil {
				return err
			}

			session.Collection("Outbox").UpdateMany(
				ctx,
				bson.M{
					"actorType": "User",
					"actorId":   user.UserID,
				},
				bson.M{
					"$set": mapof.Any{
						"actorUrl": user.ActivityPubURL(),
					},
				},
			)
		}
	}

	// Update all Stream Outbox records
	{
		cursor, err := session.Collection("Stream").Find(ctx, mapof.Any{"deleteDate": 0})

		if err != nil {
			return err
		}

		for cursor.Next(ctx) {
			stream := model.NewStream()
			if err := cursor.Decode(&stream); err != nil {
				return err
			}

			session.Collection("Outbox").UpdateMany(
				ctx,
				bson.M{
					"actorType": "Stream",
					"actorId":   stream.StreamID,
				},
				bson.M{
					"$set": mapof.Any{
						"actorUrl": stream.ActivityPubURL(),
					},
				},
			)
		}
	}

	return nil
}
