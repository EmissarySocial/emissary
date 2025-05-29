package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithParent is a Step that returns a new Stream Builder keyed to the parent of the current Stream
type WithParent struct {
	SubSteps []Step
}

// NewWithParent returns a fully initialized WithParent object
func NewWithParent(stepInfo mapof.Any) (WithParent, error) {

	const location = "build.NewWithParent"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithParent{}, derp.Wrap(err, location, "Invalid 'steps'")
	}

	return WithParent{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithParent) Name() string {
	return "with-parent"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithParent) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithParent) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
