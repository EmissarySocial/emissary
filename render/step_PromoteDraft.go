package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/first"
	"github.com/benpate/ghost/service"
)

// StepStreamPromoteDraft represents an action-step that can copy the content.Content from a StreamDraft into its corresponding Stream
type StepStreamPromoteDraft struct {
	draftService *service.StreamDraft
	stateID      string
}

func NewStepStreamPromoteDraft(draftService *service.StreamDraft, stepInfo datatype.Map) StepStreamPromoteDraft {
	return StepStreamPromoteDraft{
		draftService: draftService,
		stateID:      first.String(stepInfo.GetString("state"), "published"),
	}
}

// Get is not implemented
func (step StepStreamPromoteDraft) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post copies relevant information from the draft into the primary stream, then deletes the draft
func (step StepStreamPromoteDraft) Post(buffer io.Writer, renderer Renderer) error {

	// Try to load the draft from the database, overwriting the stream already in the renderer
	if err := step.draftService.Publish(renderer.objectID(), step.stateID); err != nil {
		return derp.Wrap(err, "ghost.renderer.StepStreamPromoteDraft.Post", "Error publishing Draft")
	}

	return nil
}
