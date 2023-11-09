package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SetFollowingCount(userCollection data.Collection, followingCollection data.Collection, userID primitive.ObjectID) error {

	criteria := exp.Equal("userId", userID).AndEqual("deleteDate", 0)
	followingCount, err := followingCollection.Count(criteria)

	if err != nil {
		return derp.Wrap(err, "queries.SetFollowingCount", "Error counting following records")
	}

	return RawUpdate(context.Background(), userCollection,
		exp.Equal("_id", userID),
		bson.M{
			"$set": bson.M{
				"followingCount": followingCount,
			},
		},
	)
}
