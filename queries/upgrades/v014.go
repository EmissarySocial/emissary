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
		err := ForEachRecord(session.Collection("Follower"), func(record mapof.Any) bool {

			if actor := record.GetMap("actor"); actor.NotEmpty() {
				if actorImageURL, ok := actor["imageUrl"]; ok {
					actor["iconUrl"] = actorImageURL
					delete(actor, "imageUrl")
					record["actor"] = actor
					return true
				}
			}

			return false
		})

		if err != nil {
			return derp.Wrap(err, "queries.upgrades.Version14", "Error updating Following collection")
		}
	}
	{
		err := ForEachRecord(session.Collection("Following"), func(record mapof.Any) bool {

			if imageURL, ok := record["imageUrl"]; ok {
				record["iconUrl"] = imageURL
				delete(record, "imageUrl")
				return true
			}

			return false
		})

		if err != nil {
			return derp.Wrap(err, "queries.upgrades.Version14", "Error updating Following collection")
		}
	}

	{
		err := ForEachRecord(session.Collection("Message"), func(record mapof.Any) bool {

			if origin := record.GetMap("origin"); origin.NotEmpty() {
				if originImageURL, ok := origin["imageUrl"]; ok {
					origin["iconUrl"] = originImageURL
					delete(origin, "imageUrl")
					record["origin"] = origin
					return true
				}
			}

			return false
		})

		if err != nil {
			return derp.Wrap(err, "queries.upgrades.Version14", "Error updating Stream collection")
		}
	}

	{
		err := ForEachRecord(session.Collection("Stream"), func(record mapof.Any) bool {

			changed := false
			if imageURL, ok := record["imageUrl"]; ok {
				record["iconUrl"] = imageURL
				delete(record, "imageUrl")
				changed = true
			}

			if attributedTo := record.GetMap("attributedTo"); attributedTo.NotEmpty() {
				if attributedToImageURL, ok := attributedTo["imageUrl"]; ok {
					attributedTo["iconUrl"] = attributedToImageURL
					delete(attributedTo, "imageUrl")
					record["attributedTo"] = attributedTo
					changed = true
				}
			}

			return changed
		})

		if err != nil {
			return derp.Wrap(err, "queries.upgrades.Version14", "Error updating Stream collection")
		}
	}

	return ForEachRecord(session.Collection("User"), func(record mapof.Any) bool {

		if imageID, ok := record["imageId"]; ok {
			record["iconId"] = imageID
			delete(record, "imageId")
			return true
		}

		return false
	})
}
