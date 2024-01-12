package model

import "github.com/benpate/rosetta/mapof"

// StateSetter wraps the SetState() method, which updates
// the state of an object.
type StateSetter interface {
	// SetState updates the state of the object. The meaning of
	// this behavior is defined by the object.
	SetState(string)
}

// RoleStateEnumerator wraps the methods required for an object
// to declare what authorized roles/state combinations are required
// for access.
type RoleStateEnumerator interface {

	// State returns the current state of the object.
	State() string

	// Roles Returns the list of roles granted by the provided authorization
	Roles(*Authorization) []string
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
