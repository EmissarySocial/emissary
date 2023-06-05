package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CountFollowers returns the total number of followers for a given user
func CountFollowers(ctx context.Context, followersCollection data.Collection, userID primitive.ObjectID) (int, error) {
	criteria := exp.Equal("parentId", userID).AndEqual("deleteDate", 0)
	return CountRecords(ctx, followersCollection, criteria)
}

func SetFollowersCount(userCollection data.Collection, followersCollection data.Collection, userID primitive.ObjectID) error {

	ctx := context.Background()
	followerCount, err := CountFollowers(ctx, followersCollection, userID)

	if err != nil {
		return derp.Wrap(err, "queries.SetFollowersCount", "Error counting followers records")
	}

	return RawUpdate(ctx, userCollection,
		exp.Equal("_id", userID),
		bson.M{
			"$set": bson.M{
				"followerCount": followerCount,
			},
		},
	)
}
