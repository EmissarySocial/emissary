package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VoteCountResult defines the results of the custom VoteCount query
type VoteCountResult struct {
	Vote  string `bson:"_id"`
	Count int    `bson:"count"`
}

// VoteCount returns the totals for all votes in the UserStream table.
func VoteCount(ctx context.Context, collection data.Collection, streamID primitive.ObjectID) ([]VoteCountResult, error) {

	result := make([]VoteCountResult, 0)

	err := pipeline(ctx, collection, result, bson.A{
		bson.M{"$match": bson.M{"_id": streamID, "vote": bson.M{"$exists": true}, "deleteDate": 0}},
		bson.M{"$group": bson.M{"_id": "$vote", "count": "$count"}},
	})

	return result, err
}

// VoteDetailResult defines the results of the custom VoteDetail query
type VoteDetailResult struct {
	Vote  string              `bson:"_id"`
	Users []model.UserSummary `bson:"users"`
}

// VoteDetail retrieves a summary of every user who voted on the designated stream -- grouped by their vote.
func VoteDetail(ctx context.Context, collection data.Collection, streamID primitive.ObjectID) ([]VoteDetailResult, error) {

	result := make([]VoteDetailResult, 0)

	err := pipeline(ctx, collection, result, bson.A{
		bson.M{"$match": bson.M{"_id": streamID, "vote": bson.M{"$exists": true}, "deleteDate": 0}},
		bson.M{"$lookup": bson.M{
			"from":         "User",
			"localField":   "userId",
			"foreignField": "_id",
		}},
	})

	return result, err
}
