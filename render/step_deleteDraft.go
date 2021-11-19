package render

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

type DeleteDraft struct {
	factory Factory
}

func NewDeleteDraft(factory Factory, config datatype.Map) DeleteDraft {
	return DeleteDraft{
		factory: factory,
	}
}

func (action DeleteDraft) Get(renderer *Renderer) error {
	return nil
}

func (action DeleteDraft) Post(renderer *Renderer) error {

	draftService := action.factory.StreamDraft()

	if err := draftService.Delete(&renderer.stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.DeleteDraft.Post", "Error deleting stream")
	}

	renderer.ctx.Response().Header().Add("hx-redirect", "/"+renderer.stream.Token)
	return renderer.ctx.NoContent(http.StatusNoContent)
}
