package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const DocumentTypeArticle = "Article"

const DocumentTypeNote = "Note"

const DocumentTypeBlock = "Block"

const DocumentTypeLike = "Like"

const DocumentTypeFollow = "Follow"

type DocumentLink struct {
	InternalID primitive.ObjectID `json:"internalId"    bson:"internalId,omitempty"` // Unique ID of a document in this database
	Author     PersonLink         `json:"author"        bson:"author,omitempty"`     // Author of this document
	URL        string             `json:"url"           bson:"url,omitempty"`        // URL of the original document
	Type       string             `json:"type"          bson:"type,omitempty"`       // ActivityStream type of document (e.g. "Article", "Note", "Image", etc.)
	Label      string             `json:"label"         bson:"label,omitempty"`      // Label/Title of the document
	Summary    string             `json:"summary"       bson:"summary,omitempty"`    // Brief summary of the document
	ImageURL   string             `json:"imageUrl"      bson:"imageUrl,omitempty"`   // URL of the cover image for this document's image
	UpdateDate int64              `json:"updateDate"    bson:"updateDate,omitempty"` // Unix timestamp of the date/time when this document was last updated
}

func NewDocumentLink() DocumentLink {
	return DocumentLink{}
}

// IsEmpty returns TRUE if this record does not link to an internal
// or external document (if both the InternalID and the URL are empty)
func (doc DocumentLink) IsEmpty() bool {
	return doc.InternalID.IsZero() && (doc.URL == "")
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

// Link returns a Link for this document
func (doc DocumentLink) Link(relation string) Link {

	return Link{
		Relation:   relation,
		InternalID: doc.InternalID,
		URL:        doc.URL,
		Label:      doc.Label,
		UpdateDate: doc.UpdateDate,
	}
}

// AuthorLink returns a correctly annotated Link to the author of this document.
func (doc DocumentLink) AuthorLink() Link {
	return doc.Author.Link(LinkRelationAuthor)
}
