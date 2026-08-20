package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxMessageSchema returns the rosetta schema that describes a OutboxMessage
func OutboxMessageSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"outboxMessageId": schema.String{Format: "objectId", Required: true},
			"actorId":         schema.String{Format: "objectId", Required: true},
			"actorType":       schema.String{Required: true, Enum: []string{FollowerTypeStream, FollowerTypeUser}},
			"activityType":    schema.String{Required: true},
			"objectId":        schema.String{Format: "url", Required: true},
			"permissions":     schema.Array{Items: schema.String{Format: "objectId"}, Required: true},
		},
	}
}

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (message *OutboxMessage) GetPointer(name string) (any, bool) {

	switch name {

	case "actorType":
		return &message.ActorType, true

	case "activityType":
		return &message.ActivityType, true

	case "objectId":
		return &message.ObjectID, true

	case "permissions":
		return &message.Permissions, true
	}

	return nil, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (message *OutboxMessage) GetStringOK(name string) (string, bool) {

	switch name {

	case "outboxMessageId":
		return message.OutboxMessageID.Hex(), true

	case "actorId":
		return message.ActorID.Hex(), true
	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
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
