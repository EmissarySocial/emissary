package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// AsModal is a Step that can update the data.DataMap custom data stored in a Stream
type AsModal struct {
	SubSteps   []Step
	Options    []string
	Background string
}

// NewAsModal returns a fully initialized AsModal object
func NewAsModal(stepInfo mapof.Any) (AsModal, error) {

	subSteps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"))

	if err != nil {
		return AsModal{}, derp.Wrap(err, "model.step.NewAsModal", "Invalid 'steps'", stepInfo)
	}

	return AsModal{
		SubSteps:   subSteps,
		Options:    stepInfo.GetSliceOfString("options"),
		Background: stepInfo.GetString("background"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step AsModal) Name() string {
	return "as-modal"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step AsModal) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step AsModal) RequiredStates() []string {
	return requiredStates(step.SubSteps...)
}

// RequiredRolesStates returns a slice of states that must be defined any Template that uses this Step
func (step AsModal) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
