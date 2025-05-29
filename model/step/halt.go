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

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Halt) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Halt) RequiredRoles() []string {
	return []string{}
}
