package step

import (
	"github.com/benpate/rosetta/mapof"
)

// RemoveEvent is a Step that removes an HX-Trigger event from the HTTP result
type RemoveEvent struct {
	Event string
}

// NewRemoveEvent returns a fully initialized RemoveEvent object
func NewRemoveEvent(stepInfo mapof.Any) (RemoveEvent, error) {

	return RemoveEvent{
		Event: stepInfo.GetString("event"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step RemoveEvent) Name() string {
	return "remove-event"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step RemoveEvent) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step RemoveEvent) RequiredRoles() []string {
	return []string{}
}
