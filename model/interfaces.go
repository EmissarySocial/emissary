package model

type RoleStateEnumerator interface {

	// State returns the current state of the object.
	State() string

	// Roles Returns the list of roles granted by the provided authorization
	Roles(*Authorization) []string
}

// Stater wraps the State() method, which lets a model object report its internal state.
// This is primarily used to determine object permissions during pipeline rendering
type Stater interface {
	State() string
}

// RoleEnumerator wraps the Roles() method, which lets a model object report the
// named roles that are authorized.  This is primarily used to determine object permissions
// during pipeline rendering.
type RoleEnumerator interface {
	// Roles returns the named roles that are allowed for the given user Authorization
	Roles(*Authorization) []string
}
