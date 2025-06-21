package queries

import (
	"context"

	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
)

func RawUpdate(ctx context.Context, collection data.Collection, criteria exp.Expression, update bson.M) error {

	// Guarantee that we're using MongoDB
	mongo := mongoCollection(collection)

	if mongo == nil {
		return derp.InternalError("queries.RawUpdate", "Collection is not a MongoDB collection")
	}

	// Update the database
	if _, err := mongo.UpdateMany(ctx, mongodb.ExpressionToBSON(criteria), update); err != nil {
		return derp.Wrap(err, "queries.RawUpdate", "Error updating records", criteria, update)
	}

	// Silence is golden.
	return nil
}
