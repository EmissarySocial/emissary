package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// SetState is a Step that can change a Stream's state
type SetState struct {
	State string
}

func NewSetState(stepInfo mapof.Any) (SetState, error) {

	stateID := stepInfo.GetString("state")

	if stateID == "" {
		return SetState{}, derp.InternalError("build.step.SetState.NewSetState", "Missing required 'state' parameter")
	}

	return SetState{
		State: stateID,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetState) Name() string {
	return "set-state"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SetState) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetState) RequiredStates() []string {
	return []string{step.State}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetState) RequiredRoles() []string {
	return []string{}
}
