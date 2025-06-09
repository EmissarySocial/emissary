package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithResponse is a Step that returns a new Response Builder
type WithResponse struct {
	SubSteps []Step
}

// NewWithResponse returns a fully initialized WithResponse object
func NewWithResponse(stepInfo mapof.Any) (WithResponse, error) {

	const location = "NewWithResponse"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithResponse{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithResponse{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithResponse) Name() string {
	return "with-response"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithResponse) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithResponse) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithResponse) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
