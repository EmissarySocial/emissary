package model

import (
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StateSetter wraps the SetState() method, which updates
// the state of an object.
type StateSetter interface {
	// SetState updates the state of the object. The meaning of
	// this behavior is defined by the object.
	SetState(string)
}

// AccessLister wraps the methods required for an object to operate
// with an ActionAccessList
type AccessLister interface {
	data.Object

	// State returns the current state of the object.
	State() string

	// IsAuthor returns TRUE if the provided UserID the author of this object
	IsAuthor(primitive.ObjectID) bool

	// IsMyself returns TRUE if this object directly represents the provided UserID
	IsMyself(primitive.ObjectID) bool

	// RolesToGroupIDs returns a map of RoleIDs to GroupIDs
	RolesToGroupIDs(...string) id.Slice

	// RolesToPrivileges returns a map of RoleIDs to Privilege strings
	RolesToPrivileges(...string) sliceof.String
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
