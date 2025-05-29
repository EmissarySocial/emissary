package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SetPassword is a Step that can update the custom data stored in a Stream
type SetPassword struct{}

// NewSetPassword returns a fully initialized SetPassword object
func NewSetPassword(stepInfo mapof.Any) (SetPassword, error) {

	return SetPassword{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetPassword) Name() string {
	return "set-password"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetPassword) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetPassword) RequiredRoles() []string {
	return []string{}
}
