package model

import (
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DocumentLink struct {
	ID           primitive.ObjectID         `json:"id,omitempty"            bson:"id,omitempty"`           // Internal ID of the record that is being linked
	URL          string                     `json:"url,omitempty"           bson:"url,omitempty"`          // URL of the original document
	Label        string                     `json:"label,omitempty"         bson:"label,omitempty"`        // Label/Title of the document
	Summary      string                     `json:"summary,omitempty"       bson:"summary,omitempty"`      // Brief summary of the document
	ImageURL     string                     `json:"imageUrl,omitempty"      bson:"imageUrl,omitempty"`     // URL of the cover image for this document's image
	AttributedTo sliceof.Object[PersonLink] `json:"attributedTo,omitempty"  bson:"attributedTo,omitempty"` // List of people who are attributed to this document
}

func NewDocumentLink() DocumentLink {
	return DocumentLink{
		AttributedTo: sliceof.NewObject[PersonLink](),
	}
}

// IsEmpty returns TRUE if this record does not link to an internal
// or external document (if both the InternalID and the URL are empty)
func (doc DocumentLink) IsEmpty() bool {
	return doc.URL == ""
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
