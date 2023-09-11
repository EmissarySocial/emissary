package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func DocumentLinkSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"id":           schema.String{Format: "objectId"},
			"url":          schema.String{Format: "url"},
			"label":        schema.String{MaxLength: 128},
			"summary":      schema.String{Format: "html"},
			"imageUrl":     schema.String{Format: "url"},
			"attributedTo": PersonLinkSchema(),
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (doc *DocumentLink) GetPointer(name string) (any, bool) {

	switch name {

	case "url":
		return &doc.URL, true

	case "label":
		return &doc.Label, true

	case "summary":
		return &doc.Summary, true

	case "imageUrl":
		return &doc.ImageURL, true

	case "attributedTo":
		return &doc.AttributedTo, true

	default:
		return "", false
	}
}

func (doc *DocumentLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "id":
		return doc.ID.Hex(), true

	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (doc *DocumentLink) SetString(name string, value string) bool {
	switch name {

	case "id":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			doc.ID = objectID
			return true
		}
	}

	return false
}
