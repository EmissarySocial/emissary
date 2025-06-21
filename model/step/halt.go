package step

import (
	"github.com/benpate/rosetta/mapof"
)

// Halt is a Step that can update the data.DataMap custom data stored in a Stream
type Halt struct{}

// NewHalt returns a fully initialized Halt object
func NewHalt(stepInfo mapof.Any) (Halt, error) {
	return Halt{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Halt) Name() string {
	return "halt"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step Halt) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Halt) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Halt) RequiredRoles() []string {
	return []string{}
}
