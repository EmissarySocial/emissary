package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
)

// SearchActivityStreamActors full-text searches the ActivityStream cache for all Actors matching the search query.
func SearchActivityStreamActors(ctx context.Context, collection data.Collection, text string) ([]model.ActorSummary, error) {

	const location = "queries.SearchActivityStreamActors"

	// Get direct access to Mongo
	mongoCollection := mongoCollection(collection)

	if mongoCollection == nil {
		return nil, derp.NewInternalError(location, "Collection is not a MongoDB collection")
	}

	// Build the query pipeline
	pipeline := []bson.M{
		{"$match": bson.M{"isActor": true, "$text": bson.M{"$search": text}}},
		{"$sort": bson.M{"score": bson.M{"$meta": "textScore"}}},
		{"$limit": 6},
		{"$replaceWith": "$object"},
		{"$project": bson.M{
			"_id":      false,
			"id":       true,
			"type":     true,
			"name":     true,
			"icon":     "$icon.href",
			"username": "$preferredUsername",
		}},
	}

	// Execute the query and return
	return Aggregate[model.ActorSummary](ctx, mongoCollection, pipeline)
}
