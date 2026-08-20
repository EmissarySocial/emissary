package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MessageMarkRead marks a single Inbox message as read, for the provided User
func MessageMarkRead(inboxCollection data.Collection, userID primitive.ObjectID, messageID primitive.ObjectID) error {

	mongo := mongoCollection(inboxCollection)

	if mongo == nil {
		return derp.Internal("queries.MessageMarkRead", "Database must be MongoDB")
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
		return derp.Wrap(err, "queries.MessageMarkRead", "Marking message read")
	}

	return nil
}
