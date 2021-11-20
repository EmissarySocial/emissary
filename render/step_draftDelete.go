package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
)

type DraftDelete struct {
	draftService *service.StreamDraft
}

func NewDraftDelete(draftService *service.StreamDraft, config datatype.Map) DraftDelete {
	return DraftDelete{
		draftService: draftService,
	}
}

func (step DraftDelete) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step DraftDelete) Post(buffer io.Writer, renderer *Renderer) error {

	if err := step.draftService.Delete(&renderer.stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.DraftDelete.Post", "Error deleting stream")
	}

	renderer.ctx.Response().Header().Add("hx-redirect", "/"+renderer.stream.Token)
	return nil
}
