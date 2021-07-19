package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// PublishDraft manages the content.Content in a stream.
type PublishDraft struct {
	factory Factory
	model.ActionConfig
}

func NewAction_PublishDraft(factory Factory, config model.ActionConfig) PublishDraft {
	return PublishDraft{
		factory:      factory,
		ActionConfig: config,
	}
}

func (action PublishDraft) Get(renderer Renderer) (string, error) {
	return "", derp.New(derp.CodeBadRequestError, "ghost.render.PublishDraft", "GET not implemented")
}

func (action PublishDraft) Post(ctx *steranko.Context, stream *model.Stream) error {

	var draft model.Stream

	// Try to load the draft from the database, overwriting the stream already in the renderer
	draftService := action.factory.StreamDraft()

	if err := draftService.LoadByID(stream.StreamID, &draft); err != nil {
		return derp.Wrap(err, "ghost.renderer.UpdateDraft.Post", "Error loading Draft")
	}

	// Try to save the draft into the Stream collection
	streamService := action.factory.Stream()

	if err := streamService.Save(&draft, ""); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PublishStreamDraft", "Error publishing draft"))
	}

	ctx.Response().Header().Add("HX-Redirect", "/"+draft.Token)
	ctx.NoContent(200)

	// Try to delete the draft... it's ok to fail silently because we have already published this to the main collection
	if err := draftService.Delete(&draft, "published"); err != nil {
		derp.Report(derp.Wrap(err, "ghost.handler.PublishStreamDraft", "Error deleting published draft"))
	}

	return nil
}
