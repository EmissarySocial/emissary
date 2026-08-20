package queries

import (
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
)

// CountResponsesByContent tallies the Responses to an object, grouped by their content, most popular first
func CountResponsesByContent(collection data.Collection, object string) (mapof.Int, error) {

	// Query pipeline to count all responses by type
	pipeline := []bson.M{
		{"$match": bson.M{"object": object}},
		{"$group": bson.M{
			"_id":   "$content",
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"count": -1}},
	}

	return GroupBy(collection, pipeline)
}
