package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithAnnotation is a Step that returns a new Rule Builder
type WithAnnotation struct {
	SubSteps []Step
}

// NewWithAnnotation returns a fully initialized WithAnnotation object
func NewWithAnnotation(stepInfo mapof.Any) (WithAnnotation, error) {

	const location = "NewWithAnnotation"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithAnnotation{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithAnnotation{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithAnnotation) Name() string {
	return "with-annotation"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithAnnotation) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithAnnotation) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithAnnotation) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
