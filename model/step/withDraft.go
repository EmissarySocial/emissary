package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithDraft is a Step that returns a new StreamDraft Builder
type WithDraft struct {
	SubSteps []Step
}

// NewWithDraft returns a fully initialized WithDraft object
func NewWithDraft(stepInfo mapof.Any) (WithDraft, error) {

	const location = "build.NewWithDraft"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithDraft{}, derp.Wrap(err, location, "Invalid 'steps'")
	}

	return WithDraft{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithDraft) Name() string {
	return "with-draft"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithDraft) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithDraft) RequiredStates() []string {
	return requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithDraft) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
