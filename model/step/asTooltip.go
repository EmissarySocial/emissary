package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// AsTooltip is an action-step that can update the data.DataMap custom data stored in a Stream
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

// AmStep is here only to verify that this struct is a build pipeline step
func (step AsTooltip) AmStep() {}
