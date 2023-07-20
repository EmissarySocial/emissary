package queries

import (
	"context"
	"math"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CountMessages returns the total number of messages for a particular user/folder/publishDate.
// This is used to assign a "rank" to each message, which is used to sort the messages in the inbox.
func CountMessages(inboxCollection data.Collection, userID primitive.ObjectID, folderID primitive.ObjectID, publishDate int64) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("folderId", folderID).AndEqual("publishDate", publishDate)
	return CountRecords(context.Background(), inboxCollection, criteria)
}

// CountUnreadMessages returns the total number of messages for a partucular user/folder
// that have not been read
func CountUnreadMessages(inboxCollection data.Collection, userID primitive.ObjectID, folderID primitive.ObjectID) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("folderId", folderID).AndEqual("readDate", math.MaxInt64)
	return CountRecords(context.Background(), inboxCollection, criteria)
}

// CountOutboxMessages returns the total number of messages for a particular user/publishDate.
// This is used to assign a "rank" to each message, which is used to sort the messages in the inbox.
func CountOutboxMessages(outboxCollection data.Collection, userID primitive.ObjectID, publishDate int64) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("publishDate", publishDate)
	return CountRecords(context.Background(), outboxCollection, criteria)
}

func MessageMarkRead(inboxCollection data.Collection, userID primitive.ObjectID, messageID primitive.ObjectID) error {

	mongo := mongoCollection(inboxCollection)

	if mongo == nil {
		return derp.NewInternalError("queries.MessageMarkRead", "Database must be MongoDB")
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
