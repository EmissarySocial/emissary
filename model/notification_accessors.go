package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NotificationSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"notificationId": schema.String{Format: "objectId"},
			"userId":         schema.String{Format: "objectId"},
			"type":           schema.String{Enum: []string{NotificationTypeDirect, NotificationTypeMention, NotificationTypeReply, NotificationTypeLike, NotificationTypeDislike, NotificationTypeAnnounce, NotificationTypeFollow}},
			"subtype":        schema.String{Enum: []string{NotificationSubtypeFollowing, NotificationSubtypeNotFollowing, NotificationSubtypeMLS, NotificationSubtypePlaintext}},
			"actor":          PersonLinkSchema(),
			"activityId":     schema.String{Format: "url"},
			"objectUrl":      schema.String{Format: "url"},
			"objectSummary":  schema.String{Format: "text", MaxLength: 1024},
			"streamId":       schema.String{Format: "objectId"},
			"inReplyTo":      schema.String{Format: "url"},
			"readDate":       schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (notification *Notification) GetPointer(name string) (any, bool) {

	switch name {

	case "actor":
		return &notification.Actor, true

	case "type":
		return &notification.Type, true

	case "subtype":
		return &notification.Subtype, true

	case "activityId":
		return &notification.ActivityID, true

	case "objectUrl":
		return &notification.ObjectURL, true

	case "objectSummary":
		return &notification.ObjectSummary, true

	case "inReplyTo":
		return &notification.InReplyTo, true

	case "readDate":
		return &notification.ReadDate, true
	}

	return nil, false
}

func (notification *Notification) GetStringOK(name string) (string, bool) {
	switch name {

	case "notificationId":
		return notification.NotificationID.Hex(), true

	case "userId":
		return notification.UserID.Hex(), true

	case "streamId":
		return notification.StreamID.Hex(), true
	}

	return "", false
}

func (notification *Notification) SetString(name string, value string) bool {
	switch name {

	case "notificationId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			notification.NotificationID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			notification.UserID = objectID
			return true
		}

	case "streamId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			notification.StreamID = objectID
			return true
		}
	}

	return false
}
