package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithCircle is a Step executes a list of sub-steps on every child of the current Stream
type WithCircle struct {
	SubSteps []Step
}

// NewWithCircle returns a fully initialized WithCircle object
func NewWithCircle(stepInfo mapof.Any) (WithCircle, error) {

	const location = "NewWithCircle"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithCircle{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithCircle{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithCircle) Name() string {
	return "with-circle"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithCircle) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithCircle) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithCircle) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
