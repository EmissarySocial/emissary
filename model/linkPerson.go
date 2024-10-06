package model

import (
	"net/url"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PersonLink struct {
	UserID       primitive.ObjectID `json:"userId,omitempty"       bson:"userId,omitempty"`       // Internal ID of the person (if they exist in this database)
	Name         string             `json:"name,omitempty"         bson:"name,omitempty"`         // Name of the person
	Username     string             `json:"username,omitempty"     bson:"username,omitempty"`     // Username of the person (e.g. @user@domain.social)
	ProfileURL   string             `json:"profileUrl,omitempty"   bson:"profileUrl,omitempty"`   // URL of the person's profile
	InboxURL     string             `json:"inboxUrl,omitempty"     bson:"inboxUrl,omitempty"`     // URL of the person's inbox
	EmailAddress string             `json:"emailAddress,omitempty" bson:"emailAddress,omitempty"` // Email address of the person
	IconURL      string             `json:"iconUrl,omitempty"      bson:"iconUrl,omitempty"`      // URL of the person's avatar/icon image
}

func NewPersonLink() PersonLink {
	return PersonLink{}
}

// IsEmpty returns TRUE if this record does not link to an internal
// or external person (if the UserID, ProfileURL, and Name are all empty)
func (person PersonLink) IsEmpty() bool {
	return person.UserID.IsZero() && (person.ProfileURL == "") && (person.Name == "")
}

// NotEmpty returns TRUE if this record is not empty.
func (person PersonLink) NotEmpty() bool {
	return !person.IsEmpty()
}

// HasIconURL returns TRUE if this person has a non-empty icon
func (person PersonLink) HasIconURL() bool {
	return person.IconURL != ""
}

// UsernameOrID returns the best-available username for this Person
func (person PersonLink) UsernameOrID() string {
	if person.Username != "" {
		return person.Username
	}

	if person.ProfileURL != "" {
		return person.ProfileURL
	}

	if person.EmailAddress != "" {
		return person.EmailAddress
	}

	return person.InboxURL
}

// GetJSONLD returns a JSON-LD representation of this Person.
func (person PersonLink) GetJSONLD() mapof.Any {

	result := mapof.Any{
		"id":   person.ProfileURL,
		"type": "Person",
	}

	if person.Name != "" {
		result["name"] = person.Name
	}

	if person.EmailAddress != "" {
		result["email"] = person.EmailAddress
	}

	if person.IconURL != "" {
		result["icon"] = person.IconURL
	}

	return result
}

// GetURL gets a named property value of this person,
// then retuns it as a parsed URL.  Only "profileUrl"
// "inboxUrl" and "iconUrl" should be passed to this
// function. all others will return nil values
func (person PersonLink) GetURL(name string) *url.URL {
	value, _ := person.GetStringOK(name)
	result, _ := url.Parse(value)
	return result
}

// PersonLinkProfileURL is a convenience function that
// returns the profile URL for a PersonLink
func PersonLinkProfileURL(person PersonLink) string {
	return person.ProfileURL
}

/******************************************
 * Map Marshalling
 ******************************************/

// MarshalMap returns a mapof.Any representation of this PersonLink
func (person PersonLink) MarshalMap() mapof.Any {
	return mapof.Any{
		"userId":       person.UserID.Hex(),
		"name":         person.Name,
		"username":     person.Username,
		"profileUrl":   person.ProfileURL,
		"inboxUrl":     person.InboxURL,
		"emailAddress": person.EmailAddress,
		"iconUrl":      person.IconURL,
	}
}

// UnmarshalMap populates this PersonLink from a mapof.Any object
func (person *PersonLink) UnmarshalMap(data mapof.Any) {
	person.UserID = objectID(data.GetString("userId"))
	person.Name = data.GetString("name")
	person.Username = data.GetString("username")
	person.ProfileURL = data.GetString("profileUrl")
	person.InboxURL = data.GetString("inboxUrl")
	person.EmailAddress = data.GetString("emailAddress")
	person.IconURL = data.GetString("iconUrl")
}

/******************************************
 * Mastodon API Methods
 ******************************************/

func (person PersonLink) Toot() object.Account {
	return object.Account{
		ID:          person.ProfileURL,
		URL:         person.ProfileURL,
		DisplayName: person.Name,
		Avatar:      person.IconURL,
	}
}
