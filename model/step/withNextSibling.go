package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithNextSibling is a Step that returns a new Stream Builder keyed to the next sibling of the current Stream
type WithNextSibling struct {
	SubSteps []Step
}

// NewWithNextSibling returns a fully initialized WithNextSibling object
func NewWithNextSibling(stepInfo mapof.Any) (WithNextSibling, error) {

	const location = "NewWithNextSibling"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithNextSibling{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithNextSibling{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithNextSibling) Name() string {
	return "with-next-sibling"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithNextSibling) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithNextSibling) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithNextSibling) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
