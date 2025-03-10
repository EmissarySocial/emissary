package queries

import (
	"context"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func LockSearchResults(ctx context.Context, collection data.Collection, searchResultIDs []primitive.ObjectID, lockID primitive.ObjectID) error {

	const location = "queries.LockSearchResults"

	// RULE: If there are no results to lock, then don't lock any results.  Duh, Karen.
	if len(searchResultIDs) == 0 {
		return nil
	}

	// Try to get the MongoDB collection from the data.Collection
	mongo := mongoCollection(collection)

	if mongo == nil {
		return derp.NewInternalError(location, "Collection is not a MongoDB Collection")
	}

	// Build the query to lock the requested SearchResults
	criteria := bson.M{
		"_id": bson.M{
			"$in": searchResultIDs,
		},
		"timeoutDate": bson.M{"$lt": time.Now().Unix()},
	}

	update := bson.M{
		"$set": bson.M{
			"lockId":      lockID,
			"timeoutDate": time.Now().Add(10 * time.Minute).Unix(),
		},
	}

	if _, err := mongo.UpdateMany(ctx, criteria, update); err != nil {
		return derp.Wrap(err, location, "Error updating search results", criteria, update)
	}

	return nil
}
