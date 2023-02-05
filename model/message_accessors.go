package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MessageSchema returns a JSON Schema that describes this object
func MessageSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"messageId":   schema.String{Format: "objectId"},
			"userId":      schema.String{Format: "objectId"},
			"origin":      OriginLinkSchema(),
			"document":    DocumentLinkSchema(),
			"contentHtml": schema.String{Format: "html"},
			"contentJson": schema.String{Format: "json"},
			"folderId":    schema.String{Format: "objectId"},
			"readDate":    schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (message *Message) GetInt64OK(name string) (int64, bool) {
	switch name {

	case "readDate":
		return message.ReadDate, true

	default:
		return 0, false
	}
}

func (message *Message) GetStringOK(name string) (string, bool) {
	switch name {

	case "messageId":
		return message.MessageID.Hex(), true

	case "userId":
		return message.UserID.Hex(), true

	case "folderId":
		return message.FolderID.Hex(), true

	case "contentHtml":
		return message.ContentHTML, true

	case "contentJson":
		return message.ContentJSON, true

	default:
		return "", false
	}
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (message *Message) SetInt64(name string, value int64) bool {
	switch name {

	case "readDate":
		message.ReadDate = value
		return true

	default:
		return false
	}
}

func (message *Message) SetString(name string, value string) bool {
	switch name {

	case "messageId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.MessageID = objectID
			return true
		}

	case "userId":

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.UserID = objectID
			return true
		}

	case "folderId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			message.FolderID = objectID
			return true
		}

	case "contentHtml":
		message.ContentHTML = value
		return true

	case "contentJson":
		message.ContentJSON = value
		return true

	}

	return false
}

/******************************************
 * Tree Traversal Methods
 ******************************************/

func (message *Message) GetObject(name string) (any, bool) {
	switch name {

	case "origin":
		return &message.Origin, true

	case "document":
		return &message.Document, true

	default:
		return nil, false
	}
}
