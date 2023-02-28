package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CountMessages returns the total number of messages for a particular user/folder/publishDate.
// This is used to assign a "rank" to each message, which is used to sort the messages in the inbox.
func CountMessages(followingCollection data.Collection, userID primitive.ObjectID, folderID primitive.ObjectID, publishDate int64) (int, error) {
	criteria := exp.Equal("userId", userID).AndEqual("folderId", folderID).AndEqual("publishDate", publishDate)
	return CountRecords(context.Background(), followingCollection, criteria)
}
