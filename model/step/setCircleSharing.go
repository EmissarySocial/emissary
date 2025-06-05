package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SetCircleSharing represents an action that can edit a top-level folder in the Domain
type SetCircleSharing struct {
	Title   string
	Message string
	Roles   []string
}

// NewSetCircleSharing returns a fully parsed SetCircleSharing object
func NewSetCircleSharing(stepInfo mapof.Any) (SetCircleSharing, error) {

	return SetCircleSharing{
		Title:   first(stepInfo.GetString("title"), "Sharing Settings"),
		Message: first(stepInfo.GetString("message"), "Determine Who Can See This Stream"),
		Roles:   stepInfo.GetSliceOfString("roles"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetCircleSharing) Name() string {
	return "set-circle-sharing"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetCircleSharing) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetCircleSharing) RequiredRoles() []string {
	return step.Roles
}
