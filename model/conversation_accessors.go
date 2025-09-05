package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ConversationSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"conversationId": schema.String{Format: "objectId"},
			"userId":         schema.String{Format: "objectId"},
			"participants":   schema.Array{Items: PersonLinkSchema()},
			"name":           schema.String{MaxLength: 128},
			"comment":        schema.String{MaxLength: 2048},
			"icon":           schema.String{MaxLength: 16},
			"stateId":        schema.String{Enum: []string{ConversationStateRead, ConversationStateUnread, ConversationStateArchived}, Required: true},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (conversation *Conversation) GetPointer(name string) (any, bool) {

	switch name {

	case "participants":
		return &conversation.Participants, true

	case "name":
		return &conversation.Name, true

	case "comment":
		return &conversation.Comment, true

	case "icon":
		return &conversation.Icon, true

	case "stateId":
		return &conversation.StateID, true
	}

	return nil, false
}

func (conversation Conversation) GetStringOK(name string) (string, bool) {

	switch name {

	case "conversationId":
		return conversation.ConversationID.Hex(), true

	case "userId":
		return conversation.UserID.Hex(), true
	}

	return "", false
}

/*********************************
 * Setter Interfaces
 *********************************/

func (conversation *Conversation) SetString(name string, value string) bool {

	switch name {

	case "conversationId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			conversation.ConversationID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			conversation.UserID = objectID
			return true
		}
	}

	return false
}
