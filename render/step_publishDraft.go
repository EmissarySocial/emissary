package render

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// PublishDraft manages the content.Content in a stream.
type PublishDraft struct {
	factory Factory
}

func NewPublishDraft(factory Factory, config datatype.Map) PublishDraft {
	return PublishDraft{
		factory: factory,
	}
}

func (action PublishDraft) Get(renderer *Renderer) error {
	return derp.New(derp.CodeBadRequestError, "ghost.render.PublishDraft", "GET not implemented")
}

func (action PublishDraft) Post(renderer *Renderer) error {

	var draft model.Stream

	// Try to load the draft from the database, overwriting the stream already in the renderer
	draftService := action.factory.StreamDraft()

	if err := draftService.LoadByID(renderer.stream.StreamID, &draft); err != nil {
		return derp.Wrap(err, "ghost.renderer.UpdateDraft.Post", "Error loading Draft")
	}

	// Try to save the draft into the Stream collection
	streamService := action.factory.Stream()

	if err := streamService.Save(&draft, ""); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PublishStreamDraft", "Error publishing draft"))
	}

	renderer.ctx.Response().Header().Add("HX-Redirect", "/"+draft.Token)
	renderer.ctx.NoContent(200)

	// Try to delete the draft... it's ok to fail silently because we have already published this to the main collection
	if err := draftService.Delete(&draft, "published"); err != nil {
		derp.Report(derp.Wrap(err, "ghost.handler.PublishStreamDraft", "Error deleting published draft"))
	}

	return nil
}
