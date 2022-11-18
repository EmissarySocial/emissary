package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PersonLink struct {
	InternalID   primitive.ObjectID `path:"internalId"   json:"internalId"   bson:"internalId,omitempty"`   // Unique ID of a document in this database
	Source       string             `path:"source"       json:"source"       bson:"source"`                 // The source that generated this document
	Relation     string             `path:"relation"     json:"relation"     bson:"relation,omitempty"`     // Relation to the person (e.g. "author", "contributor", "editor", "owner", "publisher", "webmaster")
	Name         string             `path:"name"         json:"name"         bson:"name,omitempty"`         // Name of the person
	ProfileURL   string             `path:"profileUrl"   json:"profileUrl"   bson:"profileUrl,omitempty"`   // URL of the person's profile
	EmailAddress string             `path:"emailAddress" json:"emailAddress" bson:"emailAddress,omitempty"` // Email address of the person
	ImageURL     string             `path:"photoUrl"     json:"photoUrl"     bson:"photoUrl,omitempty"`     // URL of the person's profile photo
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
			"name":         schema.String{},
			"profileUrl":   schema.String{Format: "url"},
			"emailAddress": schema.String{Format: "email"},
			"imageUrl":     schema.String{Format: "url"},
			"updateDate":   schema.Integer{},
		},
	}
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
