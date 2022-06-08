package model

type RoleStateEnumerator interface {

	// State returns the current state of the object.
	State() string

	// Roles Returns the list of roles granted by the provided authorization
	Roles(*Authorization) []string
}
