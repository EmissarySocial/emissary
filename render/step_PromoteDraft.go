package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/first"
)

// StepStreamPromoteDraft represents an action-step that can copy the Container from a StreamDraft into its corresponding Stream
type StepStreamPromoteDraft struct {
	stateID string

	BaseStep
}

func NewStepStreamPromoteDraft(stepInfo datatype.Map) (StepStreamPromoteDraft, error) {
	return StepStreamPromoteDraft{
		stateID: first.String(stepInfo.GetString("state"), "published"),
	}, nil
}

// Post copies relevant information from the draft into the primary stream, then deletes the draft
func (step StepStreamPromoteDraft) Post(factory Factory, renderer Renderer, _ io.Writer) error {

	// Try to load the draft from the database, overwriting the stream already in the renderer
	if err := factory.StreamDraft().Publish(renderer.objectID(), step.stateID); err != nil {
		return derp.Wrap(err, "renderer.StepStreamPromoteDraft.Post", "Error publishing draft")
	}

	return nil
}
