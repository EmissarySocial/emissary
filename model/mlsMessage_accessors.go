package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MLSMessageSchema returns a JSON Schema that describes this object
func MLSMessageSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"mlsMessageId": schema.String{Format: "objectId"},
			"userId":       schema.String{Format: "objectId"},
			"type":         schema.String{Enum: []string{vocab.ObjectTypeGroupInfo, vocab.ObjectTypeKeyPackage, vocab.ObjectTypePrivateMessage, vocab.ObjectTypePublicMessage, vocab.ObjectTypeWelcome}},
			"content":      schema.String{},
		},
	}
}

/******************************************
 * Getter/Setter Methods
 ******************************************/

func (message *MLSMessage) GetPointer(name string) (any, bool) {
	switch name {

	case "type":
		return &message.Type, true

	case "content":
		return &message.Content, true
	default:
		return nil, false
	}
}

func (mlsMessage *MLSMessage) GetStringOK(name string) (string, bool) {

	switch name {

	case "mlsMessageId":
		return mlsMessage.MLSMessageID.Hex(), true

	case "userId":
		return mlsMessage.UserID.Hex(), true

	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (message *MLSMessage) SetString(name string, value string) bool {

	switch name {

	case "mlsMessageId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.MLSMessageID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.UserID = objectID
			return true
		}
	}

	return false
}
