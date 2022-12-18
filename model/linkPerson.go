package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PersonLink struct {
	InternalID   primitive.ObjectID `path:"internalId"   json:"internalId"   bson:"internalId,omitempty"`   // Unique ID of a document in this database
	Name         string             `path:"name"         json:"name"         bson:"name,omitempty"`         // Name of the person
	ProfileURL   string             `path:"profileUrl"   json:"profileUrl"   bson:"profileUrl,omitempty"`   // URL of the person's profile
	InboxURL     string             `path:"inboxUrl"     json:"inboxUrl"     bson:"inboxUrl,omitempty"`     // URL of the person's inbox
	EmailAddress string             `path:"emailAddress" json:"emailAddress" bson:"emailAddress,omitempty"` // Email address of the person
	ImageURL     string             `path:"imageUrl"     json:"imageUrl"     bson:"imageUrl,omitempty"`     // URL of the person's avatar image
}

func NewPersonLink() PersonLink {
	return PersonLink{}
}

func PersonLinkSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"internalId":   schema.String{Format: "objectId"},
			"name":         schema.String{},
			"profileUrl":   schema.String{Format: "url"},
			"inboxUrl":     schema.String{Format: "url"},
			"imageUrl":     schema.String{Format: "url"},
			"emailAddress": schema.String{Format: "email"},
		},
	}
}

// IsEmpty returns TRUE if this record does not link to an internal
// or external person (if the InternalID, ProfileURL, and Name are all empty)
func (person PersonLink) IsEmpty() bool {
	return person.InternalID.IsZero() && (person.ProfileURL == "") && (person.Name == "")
}

// Link returns a Link to this person
func (person PersonLink) Link(relation string) Link {

	return Link{
		Relation:   relation,
		InternalID: person.InternalID,
		URL:        person.ProfileURL,
		Label:      person.Name,
	}
}
