package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SetRuleCount(userCollection data.Collection, ruleCollection data.Collection, userID primitive.ObjectID) error {

	// Count the rules for this User
	criteria := exp.Equal("userId", userID).AndEqual("deleteDate", 0)
	rulesCount, err := ruleCollection.Count(criteria)

	if err != nil {
		return derp.Wrap(err, "queries.SetRulesCount", "Error counting rules records")
	}

	return RawUpdate(context.TODO(), userCollection,
		exp.Equal("_id", userID),
		bson.M{
			"$set": bson.M{
				"ruleCount": rulesCount,
			},
		},
	)
}
