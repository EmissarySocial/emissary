package queries

import (
	"context"
	"time"

	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// pipeline executes a mongodb pipeline and populates the results into "result"
func pipeline(ctx context.Context, collection data.Collection, result any, pipeline bson.A, opts ...*options.AggregateOptions) error {

	const location = "queries.pipeline"

	// Guarantee that we're using MongoDB
	mongo := mongoCollection(collection)

	if mongo == nil {
		return derp.NewInternalError("queries.pipeline", "Database must be MongoDB")
	}

	// Define a cursor for the pipeline results
	cursor, err := mongo.Aggregate(ctx, pipeline, opts...)

	if err != nil {
		return derp.Wrap(err, location, "Error querying database")
	}

	// Execute the query.  Results returned in "result" pointer
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

	switch orig := original.(type) {

	case mongodb.Collection:
		return orig.Mongo()

	case *mongodb.Collection:
		return orig.Mongo()

	default:
		return nil
	}
}

func timeoutContext(seconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
}
