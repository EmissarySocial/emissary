package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepStreamPromoteDraft represents an action-step that can copy the Container from a StreamDraft into its corresponding Stream
type StepStreamPromoteDraft struct {
	StateID string
}

func (step StepStreamPromoteDraft) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepStreamPromoteDraft) UseGlobalWrapper() bool {
	return true
}

// Post copies relevant information from the draft into the primary stream, then deletes the draft
func (step StepStreamPromoteDraft) Post(renderer Renderer, _ io.Writer) error {

	factory := renderer.factory()

	// Try to load the draft from the database, overwriting the stream already in the renderer
	if err := factory.StreamDraft().Publish(renderer.objectID(), step.StateID); err != nil {
		return derp.Wrap(err, "renderer.StepStreamPromoteDraft.Post", "Error publishing draft")
	}

	return nil
}
