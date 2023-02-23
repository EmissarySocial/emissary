package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func DocumentLinkSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"internalId": schema.String{Format: "objectId"},
			"author":     PersonLinkSchema(),
			"url":        schema.String{Format: "url"},
			"type":       schema.String{},
			"label":      schema.String{MaxLength: 100},
			"summary":    schema.String{MaxLength: 1000},
			"imageUrl":   schema.String{Format: "url"},
		},
	}
}

/*********************************
 * Getter Interfaces
 *********************************/

func (doc *DocumentLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "internalId":
		return doc.InternalID.Hex(), true

	case "url":
		return doc.URL, true

	case "label":
		return doc.Label, true

	case "summary":
		return doc.Summary, true

	case "type":
		return doc.Type, true

	case "imageUrl":
		return doc.ImageURL, true

	default:
		return "", false
	}
}

/*********************************
 * Setter Interfaces
 *********************************/

func (doc *DocumentLink) SetString(name string, value string) bool {
	switch name {

	case "internalId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			doc.InternalID = objectID
			return true
		}

	case "url":
		doc.URL = value
		return true

	case "label":
		doc.Label = value
		return true

	case "summary":
		doc.Summary = value
		return true

	case "type":
		doc.Type = value
		return true

	case "imageUrl":
		doc.ImageURL = value
		return true
	}

	return false
}

/*********************************
 * Tree Traversal Methods
 *********************************/

func (doc *DocumentLink) GetObject(name string) (any, bool) {
	switch name {

	case "author":
		return &doc.Author, true

	default:
		return nil, false
	}
}
