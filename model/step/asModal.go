package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// AsModal represents an action-step that can update the data.DataMap custom data stored in a Stream
type AsModal struct {
	SubSteps []Step
}

// NewAsModal returns a fully initialized AsModal object
func NewAsModal(stepInfo datatype.Map) (AsModal, error) {

	subSteps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"))

	if err != nil {
		return AsModal{}, derp.Wrap(err, "model.step.NewAsModal", "Invalid 'steps'", stepInfo)
	}

	return AsModal{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step AsModal) AmStep() {}