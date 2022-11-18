package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DocumentLink struct {
	InternalID  primitive.ObjectID `path:"internalId"  json:"internalId"  bson:"internalId,omitempty"`  // Unique ID of a document in this database
	URL         string             `path:"url"         json:"url"         bson:"url,omitempty"`         // URL of the original document
	Label       string             `path:"label"       json:"label"       bson:"label,omitempty"`       // Label/Title of the document
	Summary     string             `path:"summary"     json:"summary"     bson:"summary,omitempty"`     // Brief summary of the document
	ImageURL    string             `path:"imageUrl"    json:"imageUrl"    bson:"imageUrl,omitempty"`    // URL of the cover image for this document's image
	ContentHTML string             `path:"contentHtml" json:"contentHtml" bson:"contentHtml,omitempty"` // HTML content of the document
	PublishDate int64              `path:"publishDate" json:"publishDate" bson:"publishDate,omitempty"` // Unix timestamp of the date/time when this document was first published
	UpdateDate  int64              `path:"updateDate"  json:"updateDate"  bson:"updateDate,omitempty"`  // Unix timestamp of the date/time when this document was last updated
}

func NewDocumentLink() DocumentLink {
	return DocumentLink{}
}

func DocumentLinkSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"internalId":  schema.String{Format: "objectId"},
			"url":         schema.String{Format: "url"},
			"label":       schema.String{},
			"summary":     schema.String{},
			"imageUrl":    schema.String{Format: "url"},
			"contentHtml": schema.String{Format: "html"},
			"publishDate": schema.Integer{},
			"updateDate":  schema.Integer{},
		},
	}
}

func (doc *DocumentLink) IsComplete() bool {

	if doc.URL == "" {
		return false
	}

	if doc.Label == "" {
		return false
	}

	if doc.Summary == "" {
		return false
	}

	if doc.ImageURL == "" {
		return false
	}

	return true
}
