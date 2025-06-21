package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// AsTooltip is a Step that can update the data.DataMap custom data stored in a Stream
type AsTooltip struct {
	SubSteps []Step
}

// NewAsTooltip returns a fully initialized AsTooltip object
func NewAsTooltip(stepInfo mapof.Any) (AsTooltip, error) {

	subSteps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"))

	if err != nil {
		return AsTooltip{}, derp.Wrap(err, "model.step.NewAsTooltip", "Invalid 'steps'", stepInfo)
	}

	return AsTooltip{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step AsTooltip) Name() string {
	return "as-tooltip"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step AsTooltip) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step AsTooltip) RequiredStates() []string {
	return requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step AsTooltip) RequiredRoles() []string {
	return requiredStates(step.SubSteps...)
}
