package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivitySchema returns a JSON Schema that describes this object
func ActivitySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"activityId":  schema.String{Format: "objectId"},
			"userId":      schema.String{Format: "objectId"},
			"origin":      OriginLinkSchema(),
			"document":    DocumentLinkSchema(),
			"contentHtml": schema.String{Format: "html"},
			"contentJson": schema.String{Format: "json"},
			"folderId":    schema.String{Format: "objectId"},
			"readDate":    schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (activity *Activity) GetInt64OK(name string) (int64, bool) {
	switch name {

	case "readDate":
		return activity.ReadDate, true

	default:
		return 0, false
	}
}

func (activity *Activity) GetStringOK(name string) (string, bool) {
	switch name {

	case "activityId":
		return activity.ActivityID.Hex(), true

	case "userId":
		return activity.UserID.Hex(), true

	case "folderId":
		return activity.FolderID.Hex(), true

	case "contentHtml":
		return activity.ContentHTML, true

	case "contentJson":
		return activity.ContentJSON, true

	default:
		return "", false
	}
}

/******************************************
 * Setter Interfaces
 ******************************************/

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

	case "userId":

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			activity.UserID = objectID
			return true
		}

	case "folderId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			activity.FolderID = objectID
			return true
		}

	case "contentHtml":
		activity.ContentHTML = value
		return true

	case "contentJson":
		activity.ContentJSON = value
		return true

	}

	return false
}

/******************************************
 * Tree Traversal Methods
 ******************************************/

func (activity *Activity) GetObject(name string) (any, bool) {
	switch name {

	case "origin":
		return &activity.Origin, true

	case "document":
		return &activity.Document, true

	default:
		return nil, false
	}
}
