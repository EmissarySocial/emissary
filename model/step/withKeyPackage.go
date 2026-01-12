package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithKeyPackage is a Step executes a list of sub-steps on every child of the current Stream
type WithKeyPackage struct {
	SubSteps []Step
}

// NewWithKeyPackage returns a fully initialized WithKeyPackage object
func NewWithKeyPackage(stepInfo mapof.Any) (WithKeyPackage, error) {

	const location = "NewWithKeyPackage"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithKeyPackage{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithKeyPackage{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithKeyPackage) Name() string {
	return "with-key-package"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithKeyPackage) RequiredModel() string {
	return "Settings"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithKeyPackage) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithKeyPackage) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
