package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SetSimpleSharing represents an action that can edit a top-level folder in the Domain
type SetSimpleSharing struct {
	Title   string
	Message string
	Roles   []string
}

// NewSetSimpleSharing returns a fully parsed SetSimpleSharing object
func NewSetSimpleSharing(stepInfo mapof.Any) (SetSimpleSharing, error) {

	return SetSimpleSharing{
		Title:   first(stepInfo.GetString("title"), "Sharing Settings"),
		Message: first(stepInfo.GetString("message"), "Determine Who Can See This Stream"),
		Roles:   stepInfo.GetSliceOfString("roles"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetSimpleSharing) Name() string {
	return "set-simple-sharing"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetSimpleSharing) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetSimpleSharing) RequiredRoles() []string {
	return step.Roles
}
