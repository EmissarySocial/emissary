// Package queries contains all of the custom queries required by this application
// that DO NOT run through the standard `data` package.  These are queries that rely
// on specific features of the database (such as mongodb aggregation, or live queries)
// that are out of scope for the data package
//
// If this application is ever migrated from mongodb, these functions will need to
// be rewritten to match the new database API.
package queries

import (
	"context"

	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// pipeline executes a mongodb pipeline and populates the results into "result"
func pipeline(ctx context.Context, collection data.Collection, result interface{}, pipeline bson.A, opts ...*options.AggregateOptions) error {

	const location = "queries.pipeline"

	// Unwrap the collection
	mongo := mongoCollection(collection)

	// Define the cursor
	cursor, err := mongo.Aggregate(ctx, pipeline, opts...)

	if err != nil {
		return derp.Wrap(err, location, "Error querying database")
	}

	// Execute the query
	if err := cursor.All(ctx, result); err != nil {
		return derp.Wrap(err, location, "Error reading results")
	}

	// Success!
	return nil
}

// mongoCollection Unwraps a data.Collection as the underlying data-mongo.Collection.
// This method is unsafe, but it *should never* fail, unless we're mid-way
// through migrating to another database.
func mongoCollection(original data.Collection) *mongo.Collection {
	return original.(*mongodb.Collection).Mongo()
}
