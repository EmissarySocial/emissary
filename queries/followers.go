package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/maps"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CountFollowers returns the total number of followers for a given user
func CountFollowers(ctx context.Context, collection data.Collection, userID primitive.ObjectID) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("journal.deleteDate", 0)
	return CountRecords(ctx, collection, criteria)
}

func SetFollowersCount(ctx context.Context, collection data.Collection, userID primitive.ObjectID) error {

	followerCount, err := CountFollowers(ctx, collection, userID)

	if err != nil {
		return derp.Wrap(err, "queries.SetFollowersCount", "Error counting followers records")
	}

	return RawUpdate(ctx, collection, exp.Equal("userId", userID), maps.Map{
		"followerCount": followerCount,
	})
}
