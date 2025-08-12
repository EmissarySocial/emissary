package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DocumentLink struct {
	ID           primitive.ObjectID // Internal ID of the record that is being linked
	URL          string             // URL of the original document
	Name         string             // Label/Title of the document
	Description  string             // Brief summary of the document
	IconURL      string             // URL of the icon image for this document
	AttributedTo PersonLink         // Person that this document is attributed to
}

func NewDocumentLink() DocumentLink {
	return DocumentLink{
		AttributedTo: NewPersonLink(),
	}
}
