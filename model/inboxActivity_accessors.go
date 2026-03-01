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
			"userId":          schema.String{Format: "objectId", Required: true},
			"type":            schema.String{Required: true},
			"activityId":      schema.String{Required: true},
			"actorId":         schema.String{Format: "url", Required: true},
			"objectId":        schema.String{Format: "url"},
			"mediaType":       schema.String{},
			"rawActivity":     schema.Object{Wildcard: schema.Any{}},
			"publishedDate":   schema.Integer{BitSize: 64, Required: true},
			"receivedDate":    schema.Integer{BitSize: 64, Required: true},
		},
	}
}

/******************************************
 * Getter/Setter Methods
 ******************************************/

func (inboxActivity *InboxActivity) GetPointer(name string) (any, bool) {
	switch name {

	case "activityId":
		return &inboxActivity.ActivityID, true

	case "type":
		return &inboxActivity.Type, true

	case "actorId":
		return &inboxActivity.ActorID, true

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
