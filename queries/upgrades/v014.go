package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version14...
func Version14(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 14")
	{
		err := ForEachRecord(session.Collection("Follower"), func(record mapof.Any) error {

			if actor := record.GetMap("actor"); actor.NotEmpty() {
				if actorImageURL, ok := actor["imageUrl"]; ok {
					actor["iconUrl"] = actorImageURL
					delete(actor, "imageUrl")
					record["actor"] = actor
				}
			}

			return nil
		})

		if err != nil {
			return derp.Wrap(err, "queries.upgrades.Version14", "Error updating Following collection")
		}
	}
	{
		err := ForEachRecord(session.Collection("Following"), func(record mapof.Any) error {

			if imageURL, ok := record["imageUrl"]; ok {
				record["iconUrl"] = imageURL
				delete(record, "imageUrl")
			}

			return nil
		})

		if err != nil {
			return derp.Wrap(err, "queries.upgrades.Version14", "Error updating Following collection")
		}
	}

	{
		err := ForEachRecord(session.Collection("Message"), func(record mapof.Any) error {

			if origin := record.GetMap("origin"); origin.NotEmpty() {
				if originImageURL, ok := origin["imageUrl"]; ok {
					origin["iconUrl"] = originImageURL
					delete(origin, "imageUrl")
					record["origin"] = origin
				}
			}

			return nil
		})

		if err != nil {
			return derp.Wrap(err, "queries.upgrades.Version14", "Error updating Stream collection")
		}
	}

	{
		err := ForEachRecord(session.Collection("Stream"), func(record mapof.Any) error {

			if imageURL, ok := record["imageUrl"]; ok {
				record["iconUrl"] = imageURL
				delete(record, "imageUrl")
			}

			if author := record.GetMap("author"); author.NotEmpty() {
				if authorImageURL, ok := author["imageUrl"]; ok {
					author["iconUrl"] = authorImageURL
					delete(author, "imageUrl")
					record["author"] = author
				}
			}

			return nil
		})

		if err != nil {
			return derp.Wrap(err, "queries.upgrades.Version14", "Error updating Stream collection")
		}
	}

	return ForEachRecord(session.Collection("User"), func(record mapof.Any) error {

		if imageID, ok := record["imageId"]; ok {
			record["iconId"] = imageID
			delete(record, "imageId")
		}

		return nil
	})

	return nil
}
