package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func AttachmentSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"attachmentId": schema.String{Format: "objectId"},
			"objectId":     schema.String{Format: "objectId"},
			"objectType":   schema.String{Enum: []string{AttachmentTypeStream, AttachmentTypeUser}},
			"original":     schema.String{},
			"rank":         schema.Integer{},
			"height":       schema.Integer{},
			"width":        schema.Integer{},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (attachment *Attachment) GetIntOK(name string) (int, bool) {

	switch name {

	case "rank":
		return attachment.Rank, true

	case "height":
		return attachment.Height, true

	case "width":
		return attachment.Width, true
	}

	return 0, false
}

func (attachment *Attachment) GetStringOK(name string) (string, bool) {

	switch name {

	case "attachmentId":
		return attachment.AttachmentID.Hex(), true

	case "objectId":
		return attachment.ObjectID.Hex(), true

	case "objectType":
		return attachment.ObjectType, true

	case "original":
		return attachment.Original, true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (attachment *Attachment) SetInt(name string, value int) bool {

	switch name {

	case "rank":
		attachment.Rank = value
		return true

	case "height":
		attachment.Height = value
		return true

	case "width":
		attachment.Width = value
		return true

	}

	return false
}

func (attachment *Attachment) SetString(name string, value string) bool {

	switch name {

	case "attachmentId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			attachment.AttachmentID = objectID
			return true
		}

	case "objectId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			attachment.ObjectID = objectID
			return true
		}

	case "objectType":
		attachment.ObjectType = value
		return true

	case "original":
		attachment.Original = value
		return true
	}

	return false
}
