package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithPrevSibling is a Step that returns a new Stream builder keyed to the previous sibling of the current stream
type WithPrevSibling struct {
	SubSteps []Step
}

// NewWithPrevSibling returns a fully initialized WithPrevSibling object
func NewWithPrevSibling(stepInfo mapof.Any) (WithPrevSibling, error) {

	const location = "NewWithPrevSibling"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithPrevSibling{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithPrevSibling{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithPrevSibling) Name() string {
	return "with-prev-sibling"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithPrevSibling) RequiredStates() []string {
	return requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithPrevSibling) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
