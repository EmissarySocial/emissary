package model

import (
	"github.com/EmissarySocial/emissary/tools/id"
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

func (activity *Activity) GetBytes(name string) []byte {
	switch name {
	case "activityId":
		return id.ToBytes(activity.ActivityID)
	case "ownerId":
		return id.ToBytes(activity.OwnerID)
	case "folderId":
		return id.ToBytes(activity.FolderID)
	default:
		return id.ToBytes(primitive.NilObjectID)
	}
}

func (activity *Activity) GetObject(name string) any {
	switch name {
	case "origin":
		return activity.Origin
	case "document":
		return activity.Document
	case "content":
		return activity.Content
	default:
		return nil
	}
}
