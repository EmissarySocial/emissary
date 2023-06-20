package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
)

// GroupBy returns amap of counts, grouped by the provided pipeline
func GroupBy(collection data.Collection, pipeline []bson.M) (mapof.Int, error) {

	mongo := mongoCollection(collection)

	if mongo == nil {
		return nil, derp.NewInternalError("queries.GroupBy", "Collection is not a MongoDB collection")
	}

	ctx := context.TODO()
	cursor, err := mongo.Aggregate(ctx, pipeline)

	if err != nil {
		return nil, derp.Wrap(err, "queries.GroupBy", "Error counting records", pipeline)
	}

	// Read results into a slice of maps
	queryResult := make([]GroupedCounter, 0)
	if err := cursor.All(ctx, &queryResult); err != nil {
		return nil, derp.Wrap(err, "queries.GroupBy", "Error reading records from cursor", pipeline)
	}

	result := mapof.NewInt()

	for _, item := range queryResult {
		result[item.Group] = item.Count
	}

	return result, nil
}
