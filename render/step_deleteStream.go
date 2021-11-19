package render

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

type DeleteStream struct {
	factory Factory
}

func NewDeleteStream(factory Factory, config datatype.Map) DeleteStream {
	return DeleteStream{
		factory: factory,
	}
}

func (action DeleteStream) Get(renderer *Renderer) error {
	return nil
}

func (action DeleteStream) Post(renderer *Renderer) error {

	var parent model.Stream

	streamService := action.factory.Stream()

	if err := streamService.LoadParent(&renderer.stream, &parent); err != nil {
		return derp.Wrap(err, "ghost.render.DeleteStream.Post", "Error loading parent stream")
	}

	if err := streamService.Delete(&renderer.stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.DeleteStream.Post", "Error deleting stream")
	}

	renderer.ctx.Response().Header().Add("hx-redirect", "/"+parent.Token)
	return renderer.ctx.NoContent(http.StatusNoContent)
}
