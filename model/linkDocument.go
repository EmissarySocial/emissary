package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DocumentLink struct {
	ID           primitive.ObjectID `json:"id,omitempty"            bson:"id,omitempty"`           // Internal ID of the record that is being linked
	URL          string             `json:"url,omitempty"           bson:"url,omitempty"`          // URL of the original document
	Label        string             `json:"label,omitempty"         bson:"label,omitempty"`        // Label/Title of the document
	Summary      string             `json:"summary,omitempty"       bson:"summary,omitempty"`      // Brief summary of the document
	IconURL      string             `json:"iconUrl,omitempty"      bson:"iconUrl,omitempty"`       // URL of the icon image for this document
	AttributedTo PersonLink         `json:"attributedTo,omitempty"  bson:"attributedTo,omitempty"` // Person that this document is attributed to
}

func NewDocumentLink() DocumentLink {
	return DocumentLink{
		AttributedTo: NewPersonLink(),
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

	if doc.IconURL == "" {
		return false
	}

	return true
}
