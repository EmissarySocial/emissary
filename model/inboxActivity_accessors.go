package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InboxActivitySchema returns a JSON Schema that describes this object
func InboxActivitySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"inboxActivityId": schema.String{Format: "objectId", Required: true},
			"actorId":         schema.String{Format: "url", Required: true},
			"userId":          schema.String{Format: "objectId", Required: true},
			"activityId":      schema.String{Required: true},
			"activityType":    schema.String{Required: true},
			"objectType":      schema.String{},
			"objectId":        schema.String{},
			"mediaType":       schema.String{},
			"rawActivity":     schema.Object{Wildcard: schema.Any{}},
			"publishedDate":   schema.Integer{BitSize: 64, Required: true},
			"receivedDate":    schema.Integer{BitSize: 64, Required: true},
			"isPublic":        schema.Boolean{},
		},
	}
}

/******************************************
 * Getter/Setter Methods
 ******************************************/

func (inboxActivity *InboxActivity) GetPointer(name string) (any, bool) {
	switch name {

	case "actorId":
		return &inboxActivity.ActorID, true

	case "activityId":
		return &inboxActivity.ActivityID, true

	case "activityType":
		return &inboxActivity.ActivityType, true

	case "objectType":
		return &inboxActivity.ObjectType, true

	case "objectId":
		return &inboxActivity.ObjectID, true

	case "mediaType":
		return &inboxActivity.MediaType, true

	case "rawActivity":
		return &inboxActivity.RawActivity, true

	case "publishedDate":
		return &inboxActivity.PublishedDate, true

	case "receivedDate":
		return &inboxActivity.ReceivedDate, true

	case "isPublic":
		return &inboxActivity.IsPublic, true

	default:
		return nil, false
	}
}

func (mlsMessage *InboxActivity) GetStringOK(name string) (string, bool) {

	switch name {

	case "inboxActivityId":
		return mlsMessage.InboxActivityID.Hex(), true

	case "userId":
		return mlsMessage.UserID.Hex(), true

	case "actorId":
		return mlsMessage.ActorID, true

	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (inboxActivity *InboxActivity) SetString(name string, value string) bool {

	switch name {

	case "inboxActivityId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			inboxActivity.InboxActivityID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			inboxActivity.UserID = objectID
			return true
		}
	}

	return false
}
