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
			"userId":          schema.String{Format: "objectId"},
			"objectType":      schema.String{},
			"objectId":        schema.String{Format: "objectId"},
			"parentId":        schema.String{Format: "objectId"},
			"document":        schema.Any{},
			"rank":            schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (message *OutboxMessage) GetInt64OK(name string) (int64, bool) {
	switch name {

	case "rank":
		return message.Rank, true

	default:
		return 0, false
	}
}

func (message *OutboxMessage) GetStringOK(name string) (string, bool) {
	switch name {

	case "outboxMessageId":
		return message.OutboxMessageID.Hex(), true

	case "userId":
		return message.UserID.Hex(), true

	case "objectType":
		return message.ObjectType, true

	case "objectId":
		return message.ObjectID.Hex(), true

	case "parentId":
		return message.ParentID.Hex(), true

	default:
		return "", false
	}
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (message *OutboxMessage) SetInt64(name string, value int64) bool {
	switch name {

	case "rank":
		message.Rank = value
		return true

	default:
		return false
	}
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

	case "objectType":
		message.ObjectType = value
		return true

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

/******************************************
 * Tree Traversal Methods
 ******************************************/

func (message *OutboxMessage) GetObject(name string) (any, bool) {
	switch name {

	case "activity":
		return &message.Activity, true

	default:
		return nil, false
	}
}
