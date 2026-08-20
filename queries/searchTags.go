package queries

import (
	"context"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"go.mongodb.org/mongo-driver/bson"
)

// SearchTags_Groups returns the distinct, non-empty group names used by SearchTags
func SearchTags_Groups(collection data.Collection) ([]string, error) {

	const location = "queries.SearchTags_Groups"

	// Get a Mongo collection
	m := mongoCollection(collection)

	if m == nil {
		return nil, derp.Internal(location, "Invalid collection")
	}

	// Context
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	criteria := bson.M{"group": bson.M{"$ne": ""}}
	sliceOfInterface, err := m.Distinct(ctx, "group", criteria)

	if err != nil {
		return nil, derp.Wrap(err, location, "Reading distinct groups")
	}

	return convert.SliceOfString(sliceOfInterface), nil
}
