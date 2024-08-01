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
			"objectType":   schema.String{Enum: []string{AttachmentObjectTypeDomain, AttachmentObjectTypeStream, AttachmentObjectTypeUser}},
			"category":     schema.String{},
			"label":        schema.String{},
			"description":  schema.String{},
			"url":          schema.String{Format: "url"},
			"original":     schema.String{},
			"status":       schema.String{Enum: []string{AttachmentStatusReady, AttachmentStatusWorking}},
			"height":       schema.Integer{},
			"width":        schema.Integer{},
			"duration":     schema.Integer{},
			"rank":         schema.Integer{},

			"rules": AttachmentRulesSchema(),
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

	case "status":
		return &attachment.Status, true

	case "rank":
		return &attachment.Rank, true

	case "height":
		return &attachment.Height, true

	case "width":
		return &attachment.Width, true

	case "duration":
		return &attachment.Duration, true

	case "rules":
		return &attachment.Rules, true
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
