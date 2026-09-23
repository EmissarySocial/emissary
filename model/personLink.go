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
	UserID       primitive.ObjectID `json:"userId,omitempty"       bson:"userId,omitempty"`       // Internal ID of the person (if they exist in this database)
	Name         string             `json:"name,omitempty"         bson:"name,omitempty"`         // Name of the person
	Username     string             `json:"username,omitempty"     bson:"username,omitempty"`     // Username of the person (e.g. @user@domain.social)
	ProfileURL   string             `json:"profileUrl,omitempty"   bson:"profileUrl,omitempty"`   // URL of the person's profile
	InboxURL     string             `json:"inboxUrl,omitempty"     bson:"inboxUrl,omitempty"`     // URL of the person's inbox
	EmailAddress string             `json:"emailAddress,omitempty" bson:"emailAddress,omitempty"` // Email address of the person
	IconURL      string             `json:"iconUrl,omitempty"      bson:"iconUrl,omitempty"`      // URL of the person's avatar/icon image
}

// NormalizeEmailAddress returns an email address in the one form Emissary stores and
// compares: trimmed and lowercased
func NormalizeEmailAddress(emailAddress string) string {
	return strings.ToLower(strings.TrimSpace(emailAddress))
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

	// Local accounts use the same short hex UserID as model.User.Toot(), so the
	// same account is identified consistently everywhere it appears (a status's
	// embedded author vs. that same account fetched directly). Remote/unlinked
	// people currently fall back to their raw profile URL; resolveAccountURL in
	// handler/mastodon/accounts.go still accepts a bare URL, so this resolves --
	// but it is inconsistent with the "u_..." token EncodeRemoteAccountID produces
	// for the same account via GetAccount_Lookup. Switching this to
	// EncodeRemoteAccountID(person.ProfileURL) closes the gap (encoding needs no
	// session/factory); left as a follow-up.
	id := person.ProfileURL

	if !person.UserID.IsZero() {
		id = person.UserID.Hex()
	}

	// The real Account entity requires a non-null created_at (confirmed against the
	// real client's Codable model -- unlike LastStatusAt, this field has no "?" and
	// crashes decode if missing/empty). ActivityPub doesn't guarantee any actor
	// publishes a reliable "account created" date (Mastodon's own Account.created_at
	// is a Mastodon-API convention, not part of ActivityPub itself, and plenty of
	// other software won't populate it), so rather than chase an inconsistently
	// available value, just use now -- an honest "we don't know" rather than a
	// crash or a lie.
	return object.Account{
		ID:          id,
		URL:         person.ProfileURL,
		Username:    person.LocalUsername(),
		Acct:        person.Username, // Already in "user" or "user@domain.social" form -- see the field comment.
		DisplayName: person.Name,
		Avatar:      person.IconURL,
		CreatedAt:   MastodonDate(time.Now()),
	}
}

// LocalUsername returns the bare username (no "@domain" suffix), for the
// Mastodon API's Account.Username field -- "not including domain," per spec,
// whereas PersonLink.Username is already qualified (e.g. "user@domain.social").
func (person PersonLink) LocalUsername() string {
	if name, _, found := strings.Cut(person.Username, "@"); found {
		return name
	}

	return person.Username
}
