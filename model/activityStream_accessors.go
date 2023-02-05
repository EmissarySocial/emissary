package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ActivityStreamSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"activityStreamId": schema.String{Format: "objectId"},
			"userId":           schema.String{Format: "objectId"},
			"publishDate":      schema.Integer{BitSize: 64},
			"container":        schema.Integer{},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (activityStream *ActivityStream) GetInt(name string) (int, bool) {
	switch name {
	case "container":
		return int(activityStream.Container), true
	}

	return 0, false
}

func (activityStream *ActivityStream) GetInt64(name string) (int64, bool) {
	switch name {
	case "publishDate":
		return activityStream.PublishDate, true
	}

	return 0, false
}

func (activityStream *ActivityStream) GetString(name string) (string, bool) {
	switch name {
	case "activityStreamId":
		return activityStream.ActivityStreamID.Hex(), true
	case "userId":
		return activityStream.UserID.Hex(), true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (activityStream *ActivityStream) SetInt(name string, value int) bool {

	switch name {

	case "container":
		activityStream.Container = ActivityStreamContainer(value)
		return true
	}

	return false
}

func (activityStream *ActivityStream) SetInt64(name string, value int64) bool {

	switch name {

	case "publishDate":
		activityStream.PublishDate = value
		return true
	}

	return false
}

func (activityStream *ActivityStream) SetString(name string, value string) bool {

	switch name {

	case "activityStreamId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			activityStream.ActivityStreamID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			activityStream.UserID = objectID
			return true
		}
	}

	return false
}
