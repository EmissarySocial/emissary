package render

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

type DeleteDraft struct {
	factory Factory
	model.ActionConfig
}

func NewAction_DeleteDraft(factory Factory, config model.ActionConfig) DeleteDraft {
	return DeleteDraft{
		factory:      factory,
		ActionConfig: config,
	}
}

func (action DeleteDraft) Get(renderer Renderer) (string, error) {
	return "", nil
}

func (action DeleteDraft) Post(ctx *steranko.Context, stream *model.Stream) error {

	draftService := action.factory.StreamDraft()

	if err := draftService.Delete(stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.DeleteDraft.Post", "Error deleting stream")
	}

	ctx.Response().Header().Add("hx-redirect", "/"+stream.Token)
	return ctx.NoContent(http.StatusNoContent)
}
