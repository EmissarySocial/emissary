package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func OutboxMessageSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"outboxMessageId": schema.String{Format: "objectId", Required: true},
			"parentType":      schema.String{Required: true, Enum: []string{FollowerTypeStream, FollowerTypeUser}},
			"parentId":        schema.String{Format: "object", Required: true},
			"activityType":    schema.String{Format: "string", Required: true},
			"url":             schema.String{Format: "url", Required: true},
		},
	}
}

func (message *OutboxMessage) GetPointer(name string) (any, bool) {
	switch name {

	case "parentType":
		return &message.ParentType, true

	case "activityType":
		return &message.ActivityType, true

	case "url":
		return &message.URL, true
	}

	return nil, false
}

func (message *OutboxMessage) GetStringOK(name string) (string, bool) {

	switch name {

	case "outboxMessageId":
		return message.OutboxMessageID.Hex(), true

	case "parentId":
		return message.ParentID.Hex(), true
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

	case "parentId":
		if objectId, err := primitive.ObjectIDFromHex(value); err == nil {
			message.ParentID = objectId
			return true
		}
	}

	return false
}
