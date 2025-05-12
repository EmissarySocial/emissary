package model

import (
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StateSetter wraps the SetState() method, which updates
// the state of an object.
type StateSetter interface {
	// SetState updates the state of the object. The meaning of
	// this behavior is defined by the object.
	SetState(string)
}

// StateGetter wraps the State() method, which returns the current state
// of an object.
type StateGetter interface {
	// State returns the current state of the object.
	State() string
}

type AuthorGetter interface {
	Author() primitive.ObjectID
}

type MyselfGetter interface {
	Myself() primitive.ObjectID
}

// FieldLister wraps the Files() method, which provides the list of fields
// to query from a database
type FieldLister interface {
	// FieldList returns the subset of fields that should be queried from the database to
	// populate this object type
	Fields() []string
}

// ActivityPubProfileGetter wraps the ActivityPubProfile() method,
// which lets a model object return its data formatted in JSON-LD
type JSONLDGetter interface {
	GetJSONLD() mapof.Any
	Created() int64
}

// WebhookDataGetter wraps the GetWebhook() method, which lets an object
// return an arbitrary data structure to be sent as a webhook
type WebhookDataGetter interface {
	GetWebhookData() mapof.Any
}
