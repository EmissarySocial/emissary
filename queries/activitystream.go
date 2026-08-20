package queries

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
)

// SearchActivityStreamActors full-text searches the ActivityStream cache for all Actors matching the search query.
func SearchActivityStreamActors(collection data.Collection, text string) ([]model.ActorSummary, error) {

	const location = "queries.SearchActivityStreamActors"

	// NILCHECK: Collection cannot be nil
	if collection == nil {
		return nil, derp.Internal(location, "Collection cannot be nil.  This should never happen.")
	}

	// Get direct access to Mongo
	mongoCollection := mongoCollection(collection)

	if mongoCollection == nil {
		return nil, derp.Internal(location, "Collection is not a MongoDB collection")
	}

	// Build the query pipeline
	pipeline := []bson.M{
		// Find all Actors that match the search text
		{"$match": bson.M{"metadata.documentCategory": "Actor", "$text": bson.M{"$search": text}}},

		// Get the top 6 most relevant results
		{"$sort": bson.M{"score": bson.M{"$meta": "textScore"}}},
		{"$limit": 6},

		// Project and simplify the results
		{"$replaceWith": "$object"},
		{"$project": bson.M{
			"_id":               false,
			"id":                true,
			"type":              true,
			"name":              true,
			"icon":              "$icon.href",
			"preferredUsername": true,
			"keyPackages":       true,
		}},

		// Sort results by name
		{"$sort": bson.M{"name": 1}},
	}

	// Execute the query and return
	return Aggregate[model.ActorSummary](collection.Context(), mongoCollection, pipeline)
}

// UpdateContext re-points every document in a collection from one conversation context to another
func UpdateContext(collection data.Collection, oldContext string, newContext string) error {

	const location = "queries.UpdateContext"

	mongoCollection := mongoCollection(collection)

	if mongoCollection == nil {
		return derp.Internal(location, "Collection is not a MongoDB collection")
	}

	// Update all documents with the old context
	_, err := mongoCollection.UpdateMany( // nolint:scopeguard
		collection.Context(),
		bson.M{"object.context": oldContext},
		bson.M{"$set": bson.M{"object.context": newContext}},
	)

	// Return errors
	if err != nil {
		return derp.Wrap(err, location, "Updating context in ActivityStream collection")
	}

	// Brilliant.
	return nil
}
