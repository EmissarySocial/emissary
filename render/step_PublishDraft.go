package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StepStreamDraftPublish represents an action-step that can copy the content.Content from a StreamDraft into its corresponding Stream
type StepStreamDraftPublish struct {
	streamService *service.Stream
	draftService  *service.StreamDraft
}

func NewStepStreamDraftPublish(streamService *service.Stream, draftService *service.StreamDraft, config datatype.Map) StepStreamDraftPublish {
	return StepStreamDraftPublish{
		streamService: streamService,
		draftService:  draftService,
	}
}

func (step StepStreamDraftPublish) Get(buffer io.Writer, renderer Renderer) error {
	return derp.New(derp.CodeBadRequestError, "ghost.render.StepStreamDraftPublish", "GET not implemented")
}

func (step StepStreamDraftPublish) Post(buffer io.Writer, renderer Renderer) error {

	streamRenderer := renderer.(Stream)
	var draft model.Stream

	// Try to load the draft from the database, overwriting the stream already in the renderer
	if err := step.draftService.LoadByID(streamRenderer.stream.StreamID, &draft); err != nil {
		return derp.Wrap(err, "ghost.renderer.StepStreamDraftPublish.Post", "Error loading Draft")
	}

	// Try to save the draft into the Stream collection
	if err := step.streamService.Save(&draft, ""); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.StepStreamDraftPublish.Post", "Error publishing draft"))
	}

	// Try to delete the draft... it's ok to fail silently because we have already published this to the main collection
	if err := step.draftService.Delete(&draft, "published"); err != nil {
		derp.Report(derp.Wrap(err, "ghost.handler.StepStreamDraftPublish.Post", "Error deleting published draft"))
	}

	streamRenderer.context().Response().Header().Add("HX-Redirect", "/"+draft.Token)

	return nil
}
