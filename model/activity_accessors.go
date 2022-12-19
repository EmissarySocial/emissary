package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*******************************************
 * Getter Methods
 *******************************************/

func (activity *Activity) GetInt64(name string) int64 {
	switch name {
	case "publishDate":
		return activity.PublishDate
	case "readDate":
		return activity.ReadDate
	default:
		return 0
	}
}

func (activity *Activity) GetObjectID(name string) primitive.ObjectID {
	switch name {
	case "activityId":
		return activity.ActivityID
	case "ownerId":
		return activity.OwnerID
	case "folderId":
		return activity.FolderID
	default:
		return primitive.NilObjectID
	}
}

func (activity *Activity) GetString(name string) string {
	switch name {
	case "originalJson":
		return activity.OriginalJSON
	default:
		return ""
	}
}
