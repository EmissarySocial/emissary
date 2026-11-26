package step

import (
	"github.com/benpate/rosetta/mapof"
)

// UnPublish is a Step that moves a Stream into a state and saves it.  The state's "outbox"
// flag decides whether followers are told the Stream is gone.
type UnPublish struct {
	StateID string // The ID of the state that this step moves the Stream into.
}

// NewUnPublish returns a fully initialized UnPublish object
func NewUnPublish(stepInfo mapof.Any) (UnPublish, error) {

	stateID := first(stepInfo.GetString("state"), "default")

	return UnPublish{
		StateID: stateID,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step UnPublish) Name() string {
	return "unpublish"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step UnPublish) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step UnPublish) RequiredStates() []string {
	return []string{step.StateID}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step UnPublish) RequiredRoles() []string {
	return []string{}
}
