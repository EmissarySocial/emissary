package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Aggregate returns amap of counts, grouped by the provided pipeline
func Aggregate[T any](ctx context.Context, collection *mongo.Collection, pipeline []bson.M, opts ...*options.AggregateOptions) ([]T, error) {

	const location = "queries.Aggregate"

	// Create a slice of results
	result := make([]T, 0)

	// Execute the mongoDB pipeline
	cursor, err := collection.Aggregate(ctx, pipeline, opts...)

	if err != nil {
		return nil, derp.ReportAndReturn(derp.Wrap(err, location, "Error counting records", pipeline))
	}

	// Read results into the result
	if err := cursor.All(ctx, &result); err != nil {
		return nil, derp.ReportAndReturn(derp.Wrap(err, location, "Error reading records from cursor", pipeline))
	}

	return result, nil
}

// GroupBy returns amap of counts, grouped by the provided pipeline
func GroupBy(collection data.Collection, pipeline []bson.M) (mapof.Int, error) {

	// Guarantee that we're using MongoDB
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
