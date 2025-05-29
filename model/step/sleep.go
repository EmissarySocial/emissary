package step

import (
	"github.com/benpate/rosetta/mapof"
)

// Sleep is a Step that sleeps for a determined amount of time.
// It should really only be used for debugging.
type Sleep struct {
	Duration int
}

// NewSleep returns a fully initialized Sleep object
func NewSleep(stepInfo mapof.Any) (Sleep, error) {

	return Sleep{
		Duration: stepInfo.GetInt("duration"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Sleep) Name() string {
	return "set-sleep"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Sleep) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Sleep) RequiredRoles() []string {
	return []string{}
}
