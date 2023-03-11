package queries

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UpdateFolderReadDate updates the "readDate" field on a folder, if the new value is greater than the existing value.
func UpdateFolderReadDate(collection data.Collection, userID primitive.ObjectID, folderID primitive.ObjectID, readDate int64) error {

	// Make sure we're using MongoDB
	mongo := mongoCollection(collection)

	if mongo == nil {
		return derp.NewInternalError("queries.UpdateFolderReadDate", "Database must be MongoDB")
	}

	// Create filter and update statements
	filter := bson.M{
		"_id":      folderID,
		"userId":   userID,
		"readDate": bson.M{"$lt": readDate},
	}

	update := bson.M{
		"$set": bson.M{
			"readDate": readDate,
		},
	}

	// Execute the conditional update
	result := mongo.FindOneAndUpdate(context.Background(), filter, update)
	resultValue := mapof.NewAny()
	result.Decode(&resultValue)

	// Woot.
	return nil
}
