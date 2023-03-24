package model

import (
	"github.com/benpate/rosetta/schema"
)

func DocumentLinkSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"url":          schema.String{Format: "url"},
			"label":        schema.String{MaxLength: 128},
			"summary":      schema.String{MaxLength: 1024},
			"imageUrl":     schema.String{Format: "url"},
			"attributedTo": schema.Array{Items: PersonLinkSchema()},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (doc *DocumentLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "url":
		return doc.URL, true

	case "label":
		return doc.Label, true

	case "summary":
		return doc.Summary, true

	case "imageUrl":
		return doc.ImageURL, true

	default:
		return "", false
	}
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (doc *DocumentLink) SetString(name string, value string) bool {
	switch name {

	case "url":
		doc.URL = value
		return true

	case "label":
		doc.Label = value
		return true

	case "summary":
		doc.Summary = value
		return true

	case "imageUrl":
		doc.ImageURL = value
		return true
	}

	return false
}

/******************************************
 * Tree Traversal Interfaces
 ******************************************/

func (doc *DocumentLink) GetObject(name string) (any, bool) {
	switch name {

	case "attributedTo":
		return &doc.AttributedTo, true
	}

	return nil, false
}
