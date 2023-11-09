package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SetBlockCount(userCollection data.Collection, blockCollection data.Collection, userID primitive.ObjectID) error {

	// Count the blocks for this User
	criteria := exp.Equal("userId", userID).AndEqual("deleteDate", 0)
	blocksCount, err := blockCollection.Count(criteria)

	if err != nil {
		return derp.Wrap(err, "queries.SetBlocksCount", "Error counting blocks records")
	}

	return RawUpdate(context.TODO(), userCollection,
		exp.Equal("_id", userID),
		bson.M{
			"$set": bson.M{
				"blockCount": blocksCount,
			},
		},
	)
}
