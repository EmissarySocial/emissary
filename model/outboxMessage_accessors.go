package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxMessageSchema returns a JSON Schema that describes this object
func OutboxMessageSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"outboxMessageId": schema.String{Format: "objectId"},
			"objectType":      schema.String{},
			"objectId":        schema.String{Format: "objectId"},
			"userId":          schema.String{Format: "objectId"},
			"parentId":        schema.String{Format: "objectId"},
			"rank":            schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (message *OutboxMessage) GetPointer(name string) (any, bool) {
	switch name {

	case "rank":
		return &message.Rank, true

	case "objectType":
		return &message.ObjectType, true
	}

	return nil, false
}

func (message *OutboxMessage) GetStringOK(name string) (string, bool) {
	switch name {

	case "outboxMessageId":
		return message.OutboxMessageID.Hex(), true

	case "userId":
		return message.UserID.Hex(), true

	case "objectId":
		return message.ObjectID.Hex(), true

	case "parentId":
		return message.ParentID.Hex(), true

	}

	return "", false
}

func (message *OutboxMessage) SetString(name string, value string) bool {
	switch name {

	case "outboxMessageId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.OutboxMessageID = objectID
			return true
		}

	case "userId":

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.UserID = objectID
			return true
		}

	case "objectId":

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.ObjectID = objectID
			return true
		}

	case "parentId":

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.ParentID = objectID
			return true
		}
	}

	return false
}
