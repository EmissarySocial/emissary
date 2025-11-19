package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version21...
func Version21(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 21")

	return ForEachRecord(session.Collection("Stream"), func(record mapof.Any) bool {

		const location = "upgrades.Version21"

		if record.GetString("navigationId") != "profile" {
			return false
		}

		attributedTo := record.GetMap("attributedTo").GetString("userId")
		attributedToID, err := primitive.ObjectIDFromHex(attributedTo)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "invalid attributedTo.userId", attributedTo))
			return false
		}

		navigationIDs := []primitive.ObjectID{attributedToID}

		if parentToken := record.GetString("parentId"); parentToken != attributedTo {
			if parentID, err := primitive.ObjectIDFromHex(parentToken); err == nil {
				navigationIDs = append(navigationIDs, parentID)
			}
		}

		record["parentIds"] = navigationIDs
		return true
	})
}
