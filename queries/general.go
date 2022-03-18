package queries

import (
	"context"

	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
)

type CountRecordsResult struct {
	Count int `bson:"count"`
}

func CountRecords(ctx context.Context, collection data.Collection, criteria exp.Expression) (int, error) {

	// Set up the mongodb pipeline query and result
	query := bson.A{
		bson.M{"$match": mongodb.ExpressionToBSON(criteria)},
		bson.M{"$count": "count"},
	}

	result := []CountRecordsResult{}

	// Try to execute the query as a mongodb pipeline
	if err := pipeline(ctx, collection, &result, query); err != nil {
		return 0, derp.Wrap(err, "queries.CountRecords", "Error counting records")
	}

	// If there are no results, then the collection is empty.
	if len(result) == 0 {
		return 0, nil
	}

	// Otherwise, return the count returned by mongo.
	return result[0].Count, nil
}
