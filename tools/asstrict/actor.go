package asstrict

import (
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
)

type Actor struct {

	// Profile
	Type              string       `json:"type" bson:"type"`
	ID                string       `json:"id" bson:"id"`
	Name              string       `json:"name" bson:"name"`
	PreferredUsername string       `json:"preferredUsername" bson:"preferredUsername"`
	Summary           string       `json:"summary" bson:"summary"`
	Image             Slice[Image] `json:"image" bson:"image"`
	Icon              Image        `json:"icon" bson:"image"`
	Tag               Slice[Tag]   `json:"tag" bson:"tag"`
	URL               string       `json:"url" bson:"url"`

	// Collections
	Attachment Slice[Attachment] `json:"attachment" bson:"attachment"`
	Inbox      string            `json:"inbox" bxon:"inbox"`
	Outbox     string            `json:"outbox" bson:"outbox"`
	Liked      string            `json:"liked" bson:"liked"`
	Featured   string            `json:"featured" bson:"featured"`
	Followers  string            `json:"followers" bson:"followers"`
	Following  string            `json:"following" bson:"following"`
	PublicKey  PublicKey         `json:"publicKey" bson:"publicKey"`
}

// Get returns a value of the given property
func (actor Actor) Get(name string) property.Value {

	switch name {

	case vocab.PropertyType:
		return property.String(actor.Type)

	case vocab.PropertyID:
		return property.String(actor.ID)

	case vocab.PropertyName:
		return property.String(actor.Name)

	case vocab.PropertyPreferredUsername:
		return property.String(actor.PreferredUsername)

	case vocab.PropertySummary:
		return property.String(actor.Summary)

	case vocab.PropertyImage:
		return actor.Image

	case vocab.PropertyIcon:
		return actor.Icon

	case vocab.PropertyTag:
		return actor.Tag

	case vocab.PropertyURL:
		return property.String(actor.URL)

	case vocab.PropertyAttachment:
		return actor.Attachment

	// Collections
	case vocab.PropertyInbox:
		return property.String(actor.Inbox)

	case vocab.PropertyOutbox:
		return property.String(actor.Outbox)

	case vocab.PropertyLiked:
		return property.String(actor.Liked)

	case vocab.PropertyFeatured:
		return property.String(actor.Featured)

	case vocab.PropertyFollowers:
		return property.String(actor.Followers)

	case vocab.PropertyFollowing:
		return property.String(actor.Following)

	case vocab.PropertyPublicKey:
		return actor.PublicKey
	}

	return property.Nil{}
}

// Set returns the value with the given property set uwu
func (actor Actor) Set(name string, value any) property.Value {

	switch name {

	case vocab.PropertyType:
		actor.Type = convert.String(value)

	case vocab.PropertyID:
		actor.ID = convert.String(value)

	case vocab.PropertyName:
		actor.Name = convert.String(value)

	case vocab.PropertyPreferredUsername:
		actor.PreferredUsername = convert.String(value)

	case vocab.PropertySummary:
		actor.Summary = convert.String(value)

	case vocab.PropertyImage:
		actor.Image = actor.Image.SetSlice(name, value)

	case vocab.PropertyIcon:
		actor.Icon = NewImage(value)

	case vocab.PropertyTag:
		actor.Tag = actor.Tag.SetSlice(name, value)

	case vocab.PropertyURL:
		actor.URL = convert.String(value)

	case vocab.PropertyAttachment:
		actor.Attachment = actor.Attachment.SetSlice(name, value)

	// Collections
	case vocab.PropertyInbox:
		actor.Inbox = convert.String(value)

	case vocab.PropertyOutbox:
		actor.Outbox = convert.String(value)

	case vocab.PropertyLiked:
		actor.Liked = convert.String(value)

	case vocab.PropertyFeatured:
		actor.Featured = convert.String(value)

	case vocab.PropertyFollowers:
		actor.Followers = convert.String(value)

	case vocab.PropertyFollowing:
		actor.Following = convert.String(value)

	case vocab.PropertyPublicKey:
		actor.PublicKey = NewPublicKey(value)
	}

	// uwu
	return actor
}

// Head returns the first value in a slices, or the value itself if it is not a slice
func (actor Actor) Head() property.Value {
	return actor
}

// Tail returns all values in a slice except the first
func (actor Actor) Tail() property.Value {
	return property.Nil{}
}

// Len returns the number of elements in the value
func (actor Actor) Len() int {
	return 1
}

// IsNil returns TRUE if the value is empty
func (actor Actor) IsNil() bool {
	return false
}

// Map returns the map representation of this value
func (actor Actor) Map() map[string]any {

	return map[string]any{
		vocab.PropertyType:              actor.Type,
		vocab.PropertyID:                actor.ID,
		vocab.PropertyName:              actor.Name,
		vocab.PropertyPreferredUsername: actor.PreferredUsername,
		vocab.PropertySummary:           actor.Summary,
		vocab.PropertyImage:             actor.Image.SliceOfMap(),
		vocab.PropertyIcon:              actor.Icon.Map(),
		vocab.PropertyTag:               actor.Tag.Map(),
		vocab.PropertyURL:               actor.URL,

		// Collections
		vocab.PropertyAttachment: actor.Attachment.SliceOfMap(),
		vocab.PropertyInbox:      actor.Inbox,
		vocab.PropertyOutbox:     actor.Outbox,
		vocab.PropertyLiked:      actor.Liked,
		vocab.PropertyFeatured:   actor.Featured,
		vocab.PropertyFollowers:  actor.Followers,
		vocab.PropertyFollowing:  actor.Following,
		vocab.PropertyPublicKey:  actor.PublicKey.Map(),
	}
}

// Raw returns the raw, unwrapped value being stored
func (actor Actor) Raw() any {
	return actor
}

// Clone returns a deep copy of a value
func (actor Actor) Clone() property.Value {
	return actor
}
