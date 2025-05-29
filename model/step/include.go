package step

import (
	"github.com/benpate/rosetta/mapof"
)

// Include is a Step that calls anoter action to continue processing
type Include struct {
	Action string
}

// NewInclude returns a fully initialized Include object
func NewInclude(stepInfo mapof.Any) (Include, error) {
	return Include{
		Action: stepInfo.GetString("action"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Include) Name() string {
	return "include"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Include) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Include) RequiredRoles() []string {
	return []string{}
}
