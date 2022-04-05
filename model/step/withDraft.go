package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// WithDraft represents an action-step that can update the data.DataMap custom data stored in a Stream
type WithDraft struct {
	SubSteps []Step
}

// NewWithDraft returns a fully initialized WithDraft object
func NewWithDraft(stepInfo datatype.Map) (WithDraft, error) {

	const location = "render.NewWithDraft"

	subSteps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"))

	if err != nil {
		return WithDraft{}, derp.Wrap(err, location, "Invalid 'steps'")
	}

	return WithDraft{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WithDraft) AmStep() {}
