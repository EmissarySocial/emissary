package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithUserConnection is a Step that scopes its sub-steps to one of the current User's external connections
type WithUserConnection struct {
	SubSteps []Step
}

// NewWithUserConnection returns a fully initialized WithUserConnection object
func NewWithUserConnection(stepInfo mapof.Any) (WithUserConnection, error) {

	const location = "model.step.NewWithUserConnection"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithUserConnection{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithUserConnection{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithUserConnection) Name() string {
	return "with-user-connection"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithUserConnection) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithUserConnection) RequiredStates() []string {
	return []string{} // states may be different in the child object
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithUserConnection) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
