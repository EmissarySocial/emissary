package queries

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SearchTagByName returns a single SearchTag, using case-insensitive matching
func SearchTagByName(collection data.Collection, name string, result *model.SearchTag) error {

	const location = "queries.SearchTagByName"

	// Mongo Collection
	m := mongoCollection(collection)

	if m == nil {
		return derp.NewInternalError(location, "Invalid collection")
	}

	// Context
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Criteria
	criteria := bson.M{
		"name":       name,
		"deleteDate": 0,
	}

	// Options
	opts := options.FindOne().
		SetCollation(&options.Collation{Locale: "en", Strength: 2})

	// Send Query
	if err := m.FindOne(ctx, criteria, opts).Decode(result); err != nil {

		if err == mongo.ErrNoDocuments {
			return derp.NewNotFoundError(location, "SearchTag not found", criteria)
		}

		return derp.Wrap(err, location, "Error reading search tag", criteria)
	}

	// Success
	return nil
}
