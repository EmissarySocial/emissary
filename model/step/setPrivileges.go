package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SetPrivileges represents an action that can edit a top-level folder in the Domain
type SetPrivileges struct {
	Title string
}

// NewSetPrivileges returns a fully parsed SetPrivileges object
func NewSetPrivileges(stepInfo mapof.Any) (SetPrivileges, error) {

	return SetPrivileges{
		Title: first(stepInfo.GetString("title"), "Product Settings"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetPrivileges) Name() string {
	return "set-privileges"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetPrivileges) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetPrivileges) RequiredRoles() []string {
	return []string{}
}
