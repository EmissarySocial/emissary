package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

type DeleteStream struct {
	streamService *service.Stream
}

func NewDeleteStream(streamService *service.Stream, config datatype.Map) DeleteStream {
	return DeleteStream{
		streamService: streamService,
	}
}

func (step DeleteStream) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step DeleteStream) Post(buffer io.Writer, renderer *Renderer) error {

	var parent model.Stream

	if err := step.streamService.LoadParent(renderer.stream, &parent); err != nil {
		return derp.Wrap(err, "ghost.render.DeleteStream.Post", "Error loading parent stream")
	}

	if err := step.streamService.Delete(renderer.stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.DeleteStream.Post", "Error deleting stream")
	}

	renderer.ctx.Response().Header().Add("hx-redirect", "/"+parent.Token)
	return nil
}
