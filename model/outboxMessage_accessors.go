package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func OutboxMessageSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"outboxMessageId": schema.String{Format: "objectId", Required: true},
			"actorId":         schema.String{Format: "object", Required: true},
			"actorType":       schema.String{Required: true, Enum: []string{FollowerTypeStream, FollowerTypeUser}},
			"activityType":    schema.String{Format: "string", Required: true},
			"objectId":        schema.String{Format: "url", Required: true},
			"publishedDate":   schema.Integer{Required: true},
			"permissions":     schema.Array{Items: schema.String{Format: "permission"}, Required: true},
		},
	}
}

func (message *OutboxMessage) GetPointer(name string) (any, bool) {

	switch name {

	case "actorType":
		return &message.ActorType, true

	case "activityType":
		return &message.ActivityType, true

	case "objectId":
		return &message.ObjectID, true

	case "publishedDate":
		return &message.CreateDate, true

	case "permissions":
		return &message.Permissions, true
	}

	return nil, false
}

func (message *OutboxMessage) GetStringOK(name string) (string, bool) {

	switch name {

	case "outboxMessageId":
		return message.OutboxMessageID.Hex(), true

	case "actorId":
		return message.ActorID.Hex(), true
	}

	return "", false
}

func (message *OutboxMessage) SetString(name string, value string) bool {

	switch name {

	case "outboxMessageId":
		if objectId, err := primitive.ObjectIDFromHex(value); err == nil {
			message.OutboxMessageID = objectId
			return true
		}

	case "actorId":
		if actorID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.ActorID = actorID
			return true
		}
	}

	return false
}
