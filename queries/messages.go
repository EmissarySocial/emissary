package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func MessageMarkRead(inboxCollection data.Collection, userID primitive.ObjectID, messageID primitive.ObjectID) error {

	mongo := mongoCollection(inboxCollection)

	if mongo == nil {
		return derp.InternalError("queries.MessageMarkRead", "Database must be MongoDB")
	}

	criteria := bson.M{
		"_id":    messageID,
		"userId": userID,
	}

	update := bson.M{
		"$set": bson.M{
			"read": true,
		},
	}

	if _, err := mongo.UpdateOne(context.Background(), criteria, update); err != nil {
		return derp.Wrap(err, "queries.MessageMarkRead", "Error marking message read")
	}

	return nil
}
