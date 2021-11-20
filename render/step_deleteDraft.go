package render

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
)

type DeleteDraft struct {
	draftService *service.StreamDraft
}

func NewDeleteDraft(draftService *service.StreamDraft, config datatype.Map) DeleteDraft {
	return DeleteDraft{
		draftService: draftService,
	}
}

func (step DeleteDraft) Get(renderer *Renderer) error {
	return nil
}

func (step DeleteDraft) Post(renderer *Renderer) error {

	if err := step.draftService.Delete(&renderer.stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.DeleteDraft.Post", "Error deleting stream")
	}

	renderer.ctx.Response().Header().Add("hx-redirect", "/"+renderer.stream.Token)
	return renderer.ctx.NoContent(http.StatusNoContent)
}
