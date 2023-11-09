package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SetFollowersCount counts the number of Followers for a specific User and updates the User record.
func SetFollowersCount(userCollection data.Collection, followersCollection data.Collection, userID primitive.ObjectID) error {

	criteria := exp.Equal("parentId", userID).AndEqual("deleteDate", 0)
	followerCount, err := followersCollection.Count(criteria)

	if err != nil {
		return derp.Wrap(err, "queries.SetFollowersCount", "Error counting followers records")
	}

	return RawUpdate(context.Background(), userCollection,
		exp.Equal("_id", userID),
		bson.M{
			"$set": bson.M{
				"followerCount": followerCount,
			},
		},
	)
}
