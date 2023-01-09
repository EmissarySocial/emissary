package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*******************************************
 * Getter Methods
 *******************************************/

func (activity *Activity) GetInt64(name string) int64 {
	switch name {
	case "readDate":
		return activity.ReadDate
	default:
		return 0
	}
}

func (activity *Activity) GetString(name string) string {
	switch name {
	case "activityId":
		return activity.ActivityID.Hex()
	case "ownerId":
		return activity.OwnerID.Hex()
	case "folderId":
		return activity.FolderID.Hex()
	default:
		return ""
	}
}

/*******************************************
 * Setter Methods
 *******************************************/

func (activity *Activity) SetInt64(name string, value int64) bool {
	switch name {

	case "readDate":
		activity.ReadDate = value
		return true

	default:
		return false
	}
}

func (activity *Activity) SetString(name string, value string) bool {
	switch name {

	case "activityId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			activity.ActivityID = objectID
			return true
		}

	case "ownerId":

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			activity.OwnerID = objectID
			return true
		}

	case "folderId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			activity.FolderID = objectID
			return true
		}
	}
	return false
}

/*******************************************
 * Tree Traversal Methods
 *******************************************/

func (activity *Activity) GetChild(name string) any {
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
