package model

import "github.com/benpate/rosetta/mapof"

type RoleStateEnumerator interface {

	// State returns the current state of the object.
	State() string

	// Roles Returns the list of roles granted by the provided authorization
	Roles(*Authorization) []string
}

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
