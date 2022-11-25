package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReplyToLink struct {
	InternalID primitive.ObjectID `path:"internalId" json:"internalId" bson:"internalId,omitempty"` // Unique ID of a document in this database
	Label      string             `path:"label"      json:"label"      bson:"label,omitempty"`      // Label of the reply
	URL        string             `path:"url"        json:"url"        bson:"url,omitempty"`        // URL of the author's profile
	UpdateDate int64              `path:"updateDate" json:"updateDate" bson:"updateDate,omitempty"` // Unix timestamp of the date/time when this author was last updated.
}

func ReplyToLinkSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"internalId": schema.String{Format: "objectId"},
			"label":      schema.String{},
			"url":        schema.String{Format: "url"},
			"updateDate": schema.Integer{},
		},
	}
}

func (replyTo ReplyToLink) IsInternal() bool {
	return !replyTo.InternalID.IsZero()

}

// IsEmpty returns TRUE if this record does not link to an internal
// or external document (if both the InternalID and the URL are empty)
func (replyTo ReplyToLink) IsEmpty() bool {
	return replyTo.InternalID.IsZero() && (replyTo.URL == "")
}

// Link returns a Link to the document that is being replied to
func (replyTo ReplyToLink) Link() Link {

	return Link{
		Relation:   LinkRelationInReplyTo,
		InternalID: replyTo.InternalID,
		URL:        replyTo.URL,
		Label:      replyTo.Label,
		UpdateDate: replyTo.UpdateDate,
	}
}
