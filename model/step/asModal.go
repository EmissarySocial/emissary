package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// AsModal represents an action-step that can update the data.DataMap custom data stored in a Stream
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

// AmStep is here only to verify that this struct is a build pipeline step
func (step AsModal) AmStep() {}
