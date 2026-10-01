package model

import (
	"net/url"
	"strings"
	"time"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PersonLink returns a PersonLink that points at this
type PersonLink struct {
	UserID       primitive.ObjectID `bson:"userId,omitempty"`       // Internal ID of the person (if they exist in this database)
	Name         string             `bson:"name,omitempty"`         // Name of the person
	Username     string             `bson:"username,omitempty"`     // Username of the person (e.g. @user@domain.social)
	ProfileURL   string             `bson:"profileUrl,omitempty"`   // URL of the person's profile
	InboxURL     string             `bson:"inboxUrl,omitempty"`     // URL of the person's inbox
	EmailAddress string             `bson:"emailAddress,omitempty"` // Email address of the person
	IconURL      string             `bson:"iconUrl,omitempty"`      // URL of the person's avatar/icon image
}

// NewPersonLink returns a fully initialized, empty PersonLink
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

// Toot returns this PersonLink as its Mastodon API equivalent
func (person PersonLink) Toot() object.Account {

	// A remote/unlinked person gets the same "u_..." token GetAccount_Lookup produces for
	// the same account, so a Status's embedded account matches that account's own GetAccount.
	id := EncodeRemoteAccountID(person.ProfileURL)

	if !person.UserID.IsZero() {
		id = person.UserID.Hex()
	}

	// created_at has no "?" in the client's Codable model and crashes decode if missing.
	// ActivityPub has no reliable "account created" date, so this is an honest "unknown".
	return object.Account{
		ID:           id,
		URL:          person.ProfileURL,
		Username:     person.LocalUsername(),
		Acct:         strings.TrimPrefix(person.Username, "@"), // "user" or "user@domain.social" -- never a leading "@".
		DisplayName:  person.Name,
		Avatar:       person.IconURL,
		AvatarStatic: person.IconURL,
		CreatedAt:    MastodonDate(time.Now()),
	}
}

// LocalUsername returns the bare username (no "@domain" suffix), for the
// Mastodon API's Account.Username field.
func (person PersonLink) LocalUsername() string {

	// RULE: Username is stored either bare ("user@domain.social") or with a leading
	// "@" (webfinger's own "acct:@user@domain" style) -- both are seen in practice.
	// Strip it before splitting, or a leading-"@" value's local part comes back
	// empty (Cut splits on the first "@", which is then the leading one itself).
	username := strings.TrimPrefix(person.Username, "@")

	if name, _, found := strings.Cut(username, "@"); found {
		return name
	}

	return username
}
