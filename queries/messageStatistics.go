package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson"
)

func CalculateMessageStatistics(ctx context.Context, collection data.Collection, message *model.Message) error {

	mongoCollection := mongoCollection(collection)

	if mongoCollection == nil {
		return derp.NewInternalError("queries.CalculateMessageStatistics", "Database must be MongoDB")
	}

	// Query the database for the number of responses of each type
	cursor, err := mongoCollection.Aggregate(ctx, ([]bson.M{
		{"$match": bson.M{"objectId": message.MessageID}},
		{"$group": bson.M{
			"_id":   "type",
			"count": "$count",
		}},
	}))

	if err != nil {
		return derp.Wrap(err, "queries.CalculateMessageStatistics", "Error running aggregate query")
	}

	// Convert the results into a map
	responseTotals := mapof.NewInt()

	for cursor.Next(ctx) {
		cursor.Decode(&responseTotals)
		spew.Dump(responseTotals)
	}

	spew.Dump(responseTotals)

	// message.ResponseTotals = responseTotals
	return nil
}
