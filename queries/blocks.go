package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/maps"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CountBlocks returns the total number of blocks for a given user
func CountBlocks(ctx context.Context, collection data.Collection, userID primitive.ObjectID) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("journal.deleteDate", 0)
	return CountRecords(ctx, collection, criteria)
}

func SetBlockCount(ctx context.Context, collection data.Collection, userID primitive.ObjectID) error {

	blocksCount, err := CountBlocks(ctx, collection, userID)

	if err != nil {
		return derp.Wrap(err, "queries.SetBlocksCount", "Error counting blocks records")
	}

	return RawUpdate(ctx, collection, exp.Equal("userId", userID), maps.Map{
		"blockCount": blocksCount,
	})
}
