package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CountFollowing returns the total number of accounts that this user is following
func CountFollowing(ctx context.Context, followingCollection data.Collection, userID primitive.ObjectID) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("deleteDate", 0)
	return CountRecords(ctx, followingCollection, criteria)
}

func SetFollowingCount(userCollection data.Collection, followingCollection data.Collection, userID primitive.ObjectID) error {

	ctx := context.Background()
	followingCount, err := CountFollowing(ctx, followingCollection, userID)

	if err != nil {
		return derp.Wrap(err, "queries.SetFollowingCount", "Error counting following records")
	}

	return RawUpdate(ctx, userCollection,
		exp.Equal("_id", userID),
		bson.M{
			"$set": bson.M{
				"followingCount": followingCount,
			},
		},
	)
}
