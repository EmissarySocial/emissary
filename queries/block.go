package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CountBlocks returns the total number of blocks for a given user
func CountBlocks(ctx context.Context, blockCollection data.Collection, userID primitive.ObjectID) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("journal.deleteDate", 0)
	return CountRecords(ctx, blockCollection, criteria)
}

func SetBlockCount(userCollection data.Collection, blockCollection data.Collection, userID primitive.ObjectID) error {

	ctx := context.Background()
	blocksCount, err := CountBlocks(ctx, blockCollection, userID)

	if err != nil {
		return derp.Wrap(err, "queries.SetBlocksCount", "Error counting blocks records")
	}

	return RawUpdate(ctx, userCollection,
		exp.Equal("_id", userID),
		bson.M{
			"$set": bson.M{
				"blockCount": blocksCount,
			},
		},
	)
}
