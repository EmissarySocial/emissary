package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/first"
	"github.com/benpate/ghost/service"
)

// StepStreamDraftPublish represents an action-step that can copy the content.Content from a StreamDraft into its corresponding Stream
type StepStreamDraftPublish struct {
	streamService *service.Stream
	draftService  *service.StreamDraft
	stateID       string
}

func NewStepStreamDraftPublish(streamService *service.Stream, draftService *service.StreamDraft, stepInfo datatype.Map) StepStreamDraftPublish {
	return StepStreamDraftPublish{
		streamService: streamService,
		draftService:  draftService,
		stateID:       first.String(stepInfo.GetString("state"), "published"),
	}
}

// Get is not implemented
func (step StepStreamDraftPublish) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post copies relevant information from the draft into the primary stream, then deletes the draft
func (step StepStreamDraftPublish) Post(buffer io.Writer, renderer Renderer) error {

	// Try to load the draft from the database, overwriting the stream already in the renderer
	if err := step.draftService.Publish(renderer.objectID(), step.stateID); err != nil {
		return derp.Wrap(err, "ghost.renderer.StepStreamDraftPublish.Post", "Error publishing Draft")
	}

	return nil
}
