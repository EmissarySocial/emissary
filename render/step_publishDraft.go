package render

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// PublishDraft manages the content.Content in a stream.
type PublishDraft struct {
	streamService *service.Stream
	draftService  *service.StreamDraft
}

func NewPublishDraft(streamService *service.Stream, draftService *service.StreamDraft, config datatype.Map) PublishDraft {
	return PublishDraft{
		streamService: streamService,
		draftService:  draftService,
	}
}

func (step PublishDraft) Get(renderer *Renderer) error {
	return derp.New(derp.CodeBadRequestError, "ghost.render.PublishDraft", "GET not implemented")
}

func (step PublishDraft) Post(renderer *Renderer) error {

	var draft model.Stream

	// Try to load the draft from the database, overwriting the stream already in the renderer
	if err := step.draftService.LoadByID(renderer.stream.StreamID, &draft); err != nil {
		return derp.Wrap(err, "ghost.renderer.UpdateDraft.Post", "Error loading Draft")
	}

	// Try to save the draft into the Stream collection
	if err := step.streamService.Save(&draft, ""); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PublishStreamDraft", "Error publishing draft"))
	}

	renderer.ctx.Response().Header().Add("HX-Redirect", "/"+draft.Token)
	renderer.ctx.NoContent(200)

	// Try to delete the draft... it's ok to fail silently because we have already published this to the main collection
	if err := step.draftService.Delete(&draft, "published"); err != nil {
		derp.Report(derp.Wrap(err, "ghost.handler.PublishStreamDraft", "Error deleting published draft"))
	}

	return nil
}
