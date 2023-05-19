package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CountResponses(parentCollection data.Collection, responseCollection data.Collection, foreignKey string, parentID primitive.ObjectID) error {

	ctx := context.TODO()

	// Get native MongoDB values for parent and response collections
	parentMongoCollection := mongoCollection(parentCollection)

	if parentMongoCollection == nil {
		return derp.NewInternalError("queries.CountResponses", "Collection is not a MongoDB collection")
	}

	responseMongoCollection := mongoCollection(responseCollection)

	if responseMongoCollection == nil {
		return derp.NewInternalError("queries.CountResponses", "Collection is not a MongoDB collection")
	}

	// Query pipeline to count all responses by type
	pipeline := []bson.M{
		{"$match": bson.M{foreignKey: parentID}},
		{"$group": bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := responseMongoCollection.Aggregate(ctx, pipeline)

	if err != nil {
		return derp.Wrap(err, "queries.CountResponses", "Error counting responses", pipeline)
	}

	// Read results into a slice of maps
	results := make([]GroupedCounter, 0)
	if err := cursor.All(ctx, &results); err != nil {
		return derp.Wrap(err, "queries.CountResponses", "Error reading responses from cursor", pipeline)
	}

	// Map the cursor into a ResponseSummary object
	responseSummary := model.NewResponseSummary()

	for _, result := range results {
		switch result.Group {
		case model.ResponseTypeLike:
			responseSummary.LikeCount += result.Count

		case model.ResponseTypeDislike:
			responseSummary.DislikeCount += result.Count

		case model.ResponseTypeMention:
			responseSummary.MentionCount += result.Count

		case model.ResponseTypeReply:
			responseSummary.ReplyCount += result.Count
		}
	}

	// Update the parent document in place...
	updateDocument := bson.M{
		"$set": bson.M{
			"responses": responseSummary,
		},
	}

	if _, err := parentMongoCollection.UpdateByID(ctx, parentID, updateDocument); err != nil {
		return derp.Wrap(err, "queries.CountResponses", "Error updating parent document")
	}

	return nil
}
