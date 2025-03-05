package queries

import (
	"context"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func LockSearchResults(ctx context.Context, collection data.Collection, searchResultIDs []primitive.ObjectID) error {

	const location = "queries.LockSearchResults"

	mongo := mongoCollection(collection)

	if mongo == nil {
		return derp.NewInternalError(location, "Collection is not a MongoDB Collection")
	}

	criteria := bson.M{
		"_id": bson.M{
			"$in":    searchResultIDs,
			"lockId": primitive.NilObjectID,
		},
	}

	update := bson.M{
		"$set": bson.M{
			"lockId":      primitive.NewObjectID(),
			"timeoutDate": time.Now().Add(10 * time.Minute).Unix(),
		},
	}

	if _, err := mongo.UpdateMany(ctx, criteria, update); err != nil {
		return derp.Wrap(err, location, "Error updating search results", criteria, update)
	}

	return nil
}
