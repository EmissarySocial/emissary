package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FolderSetUnreadCount updates the "readDate" field on a folder, if the new value is greater than the existing value.
func FolderSetUnreadCount(collection data.Collection, userID primitive.ObjectID, folderID primitive.ObjectID, unreadCount int) error {

	// Guarantee that we're using MongoDB
	mongo := mongoCollection(collection)

	if mongo == nil {
		return derp.InternalError("queries.FolderSetUnreadCount", "Database must be MongoDB")
	}

	// Create filter and update statements
	filter := bson.M{
		"_id":    folderID,
		"userId": userID,
	}

	update := bson.M{
		"$set": bson.M{
			"unreadCount": unreadCount,
		},
	}

	// Execute the conditional update
	result := mongo.FindOneAndUpdate(context.Background(), filter, update)

	if err := result.Err(); err != nil {
		return derp.Wrap(err, "queries.FolderSetUnreadCount", "Error updating folder read date", userID, folderID, unreadCount)
	}

	// Woot.
	return nil
}
