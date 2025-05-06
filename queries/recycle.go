package queries

import (
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
)

// Recycle deletes all records from the specified collection
// that were "soft deleted" more than 30 days ago.
func Recycle(session data.Session, collectionName string) error {

	const location = "queries.Recycle"

	// Get a MongoDB collection
	collection := mongoCollection(session.Collection(collectionName))

	if collection == nil {
		return derp.InternalError(location, "Collection must be a MongoDB collection")
	}

	// Set a max timeout of 60 seconds to run this query
	timeout, cancel := timeoutContext(60)
	defer cancel()

	// Delete all records that were deleted more than 30 days ago
	filter := bson.M{
		"deleteDate": bson.M{"$ne": 0},
	}

	if _, err := collection.DeleteMany(timeout, filter); err != nil {
		return derp.Wrap(err, location, "Error deleting records", filter)
	}

	// Done.
	return nil
}
