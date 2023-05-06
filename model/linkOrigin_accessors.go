package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OriginLinkSchema returns a JSON Schema for OriginLink structures
func OriginLinkSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"internalId": schema.String{Format: "objectId"},
			"type":       schema.String{Enum: []string{OriginTypeActivityPub, OriginTypeInternal, OriginTypePoll, OriginTypeRSSCloud, OriginTypeWebMention}},
			"url":        schema.String{Format: "url"},
			"label":      schema.String{MaxLength: 128},
			"summary":    schema.String{MaxLength: 1024},
			"imageUrl":   schema.String{Format: "url"},
		},
	}
}

/*********************************
 * Getter Interfaces
 *********************************/

func (origin *OriginLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "internalId":
		return origin.InternalID.Hex(), true

	case "type":
		return origin.Type, true

	case "url":
		return origin.URL, true

	case "label":
		return origin.Label, true

	case "summary":
		return origin.Summary, true

	case "imageUrl":
		return origin.ImageURL, true

	}

	return "", false
}

/*********************************
 * Setter Interfaces
 *********************************/

func (origin *OriginLink) SetString(name string, value string) bool {
	switch name {

	case "internalId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			origin.InternalID = objectID
			return true
		}

	case "type":
		origin.Type = value
		return true

	case "url":
		origin.URL = value
		return true

	case "label":
		origin.Label = value
		return true

	case "summary":
		origin.Summary = value
		return true

	case "imageUrl":
		origin.ImageURL = value
		return true
	}

	return false
}
