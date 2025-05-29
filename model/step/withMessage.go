package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithMessage is a Step that returns a new Message Builder
type WithMessage struct {
	SubSteps []Step
}

// NewWithMessage returns a fully initialized WithMessage object
func NewWithMessage(stepInfo mapof.Any) (WithMessage, error) {

	const location = "NewWithMessage"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithMessage{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithMessage{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithMessage) Name() string {
	return "with-message"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithMessage) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithMessage) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
