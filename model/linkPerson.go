package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PersonLink struct {
	InternalID   primitive.ObjectID `path:"internalId"   json:"internalId"   bson:"internalId,omitempty"`   // Unique ID of a document in this database
	Source       string             `path:"source"       json:"source"       bson:"source,omitempty"`       // The source that generated this document
	Relation     string             `path:"relation"     json:"relation"     bson:"relation,omitempty"`     // Relation to the person (e.g. "author", "contributor", "editor", "me", "owner", "publisher", "webmaster")
	Organization string             `path:"organization" json:"organization" bson:"organization,omitempty"` // Organization that this person is associated with
	Name         string             `path:"name"         json:"name"         bson:"name,omitempty"`         // Name of the person
	ProfileURL   string             `path:"profileUrl"   json:"profileUrl"   bson:"profileUrl,omitempty"`   // URL of the person's profile
	EmailAddress string             `path:"emailAddress" json:"emailAddress" bson:"emailAddress,omitempty"` // Email address of the person
	ImageURL     string             `path:"imageUrl"     json:"imageUrl"     bson:"imageUrl,omitempty"`     // URL of the person's avatar image
	UpdateDate   int64              `path:"updateDate"   json:"updateDate"   bson:"updateDate,omitempty"`   // Unix timestamp of the date/time when this person was last updated.
}

func NewPersonLink() PersonLink {
	return PersonLink{}
}

func PersonLinkSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"internalId":   schema.String{Format: "objectId"},
			"source":       schema.String{Enum: []string{LinkSourceActivityPub, LinkSourceInternal, LinkSourceRSS, LinkSourceTwitter}},
			"relation":     schema.String{Enum: []string{LinkRelationAuthor, LinkRelationProfile}},
			"organization": schema.String{},
			"name":         schema.String{},
			"profileUrl":   schema.String{Format: "url"},
			"imageUrl":     schema.String{Format: "url"},
			"emailAddress": schema.String{Format: "email"},
			"updateDate":   schema.Integer{},
		},
	}
}

// IsEmpty returns TRUE if this record does not link to an internal
// or external person (if the InternalID, ProfileURL, and Name are all empty)
func (person PersonLink) IsEmpty() bool {
	return person.InternalID.IsZero() && (person.ProfileURL == "") && (person.Name == "")
}

// Link returns a Link to this person
func (person PersonLink) Link() Link {

	return Link{
		Relation:   person.Relation,
		InternalID: person.InternalID,
		URL:        person.ProfileURL,
		Label:      person.Name,
		UpdateDate: person.UpdateDate,
	}
}
