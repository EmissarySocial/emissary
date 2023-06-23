package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CountMessages returns the total number of messages for a particular user/folder/publishDate.
// This is used to assign a "rank" to each message, which is used to sort the messages in the inbox.
func CountMessages(inboxCollection data.Collection, userID primitive.ObjectID, folderID primitive.ObjectID, publishDate int64) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("folderId", folderID).AndEqual("publishDate", publishDate)
	return CountRecords(context.Background(), inboxCollection, criteria)
}

// CountMessagesAfterRank returns the total number of messages for a partucular user/folder
// that have a rank greater than the specified value.
func CountMessagesAfterRank(inboxCollection data.Collection, userID primitive.ObjectID, folderID primitive.ObjectID, minRank int64) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("folderId", folderID).AndGreaterThan("rank", minRank)
	return CountRecords(context.Background(), inboxCollection, criteria)
}

// CountOutboxMessages returns the total number of messages for a particular user/publishDate.
// This is used to assign a "rank" to each message, which is used to sort the messages in the inbox.
func CountOutboxMessages(outboxCollection data.Collection, userID primitive.ObjectID, publishDate int64) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("publishDate", publishDate)
	return CountRecords(context.Background(), outboxCollection, criteria)
}
