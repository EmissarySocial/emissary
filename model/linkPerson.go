package model

import (
	"net/url"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PersonLink struct {
	InternalID   primitive.ObjectID `json:"internalId"   bson:"internalId,omitempty"`   // Unique ID of a document in this database
	Name         string             `json:"name"         bson:"name,omitempty"`         // Name of the person
	ProfileURL   string             `json:"profileUrl"   bson:"profileUrl,omitempty"`   // URL of the person's profile
	InboxURL     string             `json:"inboxUrl"     bson:"inboxUrl,omitempty"`     // URL of the person's inbox
	EmailAddress string             `json:"emailAddress" bson:"emailAddress,omitempty"` // Email address of the person
	ImageURL     string             `json:"imageUrl"     bson:"imageUrl,omitempty"`     // URL of the person's avatar image
}

func NewPersonLink() PersonLink {
	return PersonLink{}
}

// IsEmpty returns TRUE if this record does not link to an internal
// or external person (if the InternalID, ProfileURL, and Name are all empty)
func (person PersonLink) IsEmpty() bool {
	return person.InternalID.IsZero() && (person.ProfileURL == "") && (person.Name == "")
}

func (person PersonLink) GetJSONLD() mapof.Any {

	result := mapof.Any{
		"@id":   person.ProfileURL,
		"@type": "Person",
		"id":    person.ProfileURL,
		"type":  "Person",
	}

	if person.Name != "" {
		result["name"] = person.Name
	}

	if person.EmailAddress != "" {
		result["email"] = person.EmailAddress
	}

	if person.ImageURL != "" {
		result["image"] = person.ImageURL
	}

	return result
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

// GetURL gets a named property value of this person,
// then retuns it as a parsed URL.  Only "profileUrl"
// "inboxUrl" and "imageUrl" should be passed to this
// function. all others will return nil values
func (person PersonLink) GetURL(name string) *url.URL {
	value, _ := person.GetStringOK(name)
	result, _ := url.Parse(value)
	return result
}
