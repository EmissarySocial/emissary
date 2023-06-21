package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepStreamPromoteDraft represents an action-step that can copy the Container from a StreamDraft into its corresponding Stream
type StepStreamPromoteDraft struct {
	StateID string
}

func (step StepStreamPromoteDraft) Get(renderer Renderer, _ io.Writer) ExitCondition {
	return nil
}

// Post copies relevant information from the draft into the primary stream, then deletes the draft
func (step StepStreamPromoteDraft) Post(renderer Renderer, _ io.Writer) ExitCondition {

	streamRenderer := renderer.(*Stream)

	factory := renderer.factory()

	// Try to load the draft from the database, overwriting the stream already in the renderer
	stream, err := factory.StreamDraft().Promote(renderer.objectID(), step.StateID)

	if err != nil {
		return ExitError(derp.Wrap(err, "renderer.StepStreamPromoteDraft.Post", "Error publishing draft"))
	}

	// Push the newly updated stream back to the renderer so that subsequent
	// steps (e.g. publish) can use the correct data.
	streamRenderer.stream = &stream

	return nil
}
