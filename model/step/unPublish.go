package step

import (
	"github.com/benpate/rosetta/mapof"
)

// UnPublish is a Step that can update a stream's UnPublishDate with the current time.
type UnPublish struct {
	Outbox  bool
	StateID string
}

// NewUnPublish returns a fully initialized UnPublish object
func NewUnPublish(stepInfo mapof.Any) (UnPublish, error) {

	stateID := first(stepInfo.GetString("state"), "default")

	return UnPublish{
		Outbox:  stepInfo.GetBool("outbox"),
		StateID: stateID,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step UnPublish) Name() string {
	return "unpublish"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step UnPublish) RequiredStates() []string {
	return []string{step.StateID}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step UnPublish) RequiredRoles() []string {
	return []string{}
}
