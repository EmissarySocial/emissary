package queries

import (
	"context"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"go.mongodb.org/mongo-driver/bson"
)

func SearchTags_Groups(collection data.Collection) ([]string, error) {

	const location = "queries.SearchTags_Groups"

	// Get a Mongo collection
	m := mongoCollection(collection)

	if m == nil {
		return nil, derp.NewInternalError(location, "Invalid collection")
	}

	// Context
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	criteria := bson.M{"group": bson.M{"$ne": ""}}
	sliceOfInterface, err := m.Distinct(ctx, "group", criteria)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error reading distinct groups")
	}

	return convert.SliceOfString(sliceOfInterface), nil
}
