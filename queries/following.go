package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/maps"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CountFollowing returns the total number of accounts that this user is following
func CountFollowing(ctx context.Context, collection data.Collection, userID primitive.ObjectID) (int, error) {

	criteria := exp.Equal("userId", userID).AndEqual("journal.deleteDate", 0)

	return CountRecords(ctx, collection, criteria)
}

func SetFollowingCount(ctx context.Context, collection data.Collection, userID primitive.ObjectID) error {

	followingCount, err := CountFollowing(ctx, collection, userID)

	if err != nil {
		return derp.Wrap(err, "queries.SetFollowingCount", "Error counting following records")
	}

	return RawUpdate(ctx, collection, exp.Equal("userId", userID), maps.Map{
		"followingCount": followingCount,
	})
}
