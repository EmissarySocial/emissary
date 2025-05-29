package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithChildren is a Step executes a list of sub-steps on every child of the current Stream
type WithChildren struct {
	SubSteps []Step
}

// NewWithChildren returns a fully initialized WithChildren object
func NewWithChildren(stepInfo mapof.Any) (WithChildren, error) {

	const location = "NewWithChildren"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithChildren{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithChildren{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithChildren) Name() string {
	return "with-children"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithChildren) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithChildren) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
