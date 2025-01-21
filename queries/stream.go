package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MaxRankResult defines the results of the custom MaxRank query
type MaxRankResult struct {
	MaxRank int `bson:"maxRank"`
}

// MaxRank returns the maximum rank of all children of the parent stream
func MaxRank(ctx context.Context, collection data.Collection, parentID primitive.ObjectID) (int, error) {

	// Set up the mongodb pipeline query and result
	query := bson.A{
		bson.M{"$match": bson.M{"parentId": parentID, "deleteDate": 0}},
		bson.M{"$group": bson.M{"_id": nil, "maxRank": bson.M{"$max": "$rank"}}},
	}

	result := []MaxRankResult{}

	// Try to execute the query as a mongodb pipeline
	if err := pipeline(ctx, collection, &result, query); err != nil {
		return 0, derp.Wrap(err, "queries.CountRecords", "Error counting records")
	}

	// If there are no results, then the collection is empty.
	if len(result) == 0 {
		return 0, nil
	}

	// Otherwise, return the count returned by mongo.
	return result[0].MaxRank + 1, nil
}

func SetAttributedTo(ctx context.Context, collection data.Collection, personLink model.PersonLink) error {

	criteria := exp.Equal("attributedTo.userId", personLink.UserID)

	update := bson.M{
		"$set": bson.M{
			"attributedTo": personLink,
		},
	}

	return RawUpdate(ctx, collection, criteria, update)
}
