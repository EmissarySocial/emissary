package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithAttachment is a Step that returns a new Attachment Builder
type WithAttachment struct {
	SubSteps []Step
}

// NewWithAttachment returns a fully initialized WithAttachment object
func NewWithAttachment(stepInfo mapof.Any) (WithAttachment, error) {

	const location = "NewWithAttachment"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithAttachment{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithAttachment{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithAttachment) Name() string {
	return "with-attachment"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithAttachment) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithAttachment) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithAttachment) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
