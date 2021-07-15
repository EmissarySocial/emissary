package render

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

type DeleteStream struct {
	factory Factory
	model.ActionConfig
}

func NewAction_DeleteStream(factory Factory, config model.ActionConfig) DeleteStream {
	return DeleteStream{
		factory:      factory,
		ActionConfig: config,
	}
}

func (action DeleteStream) Get(renderer Renderer) (string, error) {
	return "", nil
}

func (action DeleteStream) Post(ctx *steranko.Context, stream *model.Stream) error {

	var parent model.Stream

	streamService := action.factory.Stream()

	if err := streamService.LoadParent(stream, &parent); err != nil {
		return derp.Wrap(err, "ghost.render.DeleteStream.Post", "Error loading parent stream")
	}

	if err := streamService.Delete(stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.DeleteStream.Post", "Error deleting stream")
	}

	ctx.Response().Header().Add("hx-redirect", "/"+parent.Token)
	return ctx.NoContent(http.StatusNoContent)
}
