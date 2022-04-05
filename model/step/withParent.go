package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// WithParent represents an action-step that can update the data.DataMap custom data stored in a Stream
type WithParent struct {
	SubSteps []Step
}

// NewWithParent returns a fully initialized WithParent object
func NewWithParent(stepInfo datatype.Map) (WithParent, error) {

	const location = "render.NewWithParent"

	subSteps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"))

	if err != nil {
		return WithParent{}, derp.Wrap(err, location, "Invalid 'steps'")
	}

	return WithParent{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WithParent) AmStep() {}
