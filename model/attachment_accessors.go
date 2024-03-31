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
			"objectType":   schema.String{Enum: []string{AttachmentObjectTypeStream, AttachmentObjectTypeUser}},
			"mediaType":    schema.String{Enum: []string{AttachmentMediaTypeAny, AttachmentMediaTypeAudio, AttachmentMediaTypeDocument, AttachmentMediaTypeImage, AttachmentMediaTypeVideo}},
			"category":     schema.String{},
			"label":        schema.String{},
			"description":  schema.String{},
			"url":          schema.String{Format: "url"},
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

func (attachment *Attachment) GetPointer(name string) (any, bool) {

	switch name {

	case "objectType":
		return &attachment.ObjectType, true

	case "mediaType":
		return &attachment.MediaType, true

	case "category":
		return &attachment.Category, true

	case "label":
		return &attachment.Label, true

	case "description":
		return &attachment.Description, true

	case "url":
		return &attachment.URL, true

	case "original":
		return &attachment.Original, true

	case "rank":
		return &attachment.Rank, true

	case "height":
		return &attachment.Height, true

	case "width":
		return &attachment.Width, true
	}

	return "", false
}

func (attachment *Attachment) GetStringOK(name string) (string, bool) {

	switch name {

	case "attachmentId":
		return attachment.AttachmentID.Hex(), true

	case "objectId":
		return attachment.ObjectID.Hex(), true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

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
	}

	return false
}
