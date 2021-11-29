package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StepStreamDelete represents an action-step that can delete a Stream from the Domain
type StepStreamDelete struct {
	streamService *service.Stream
	draftService  *service.StreamDraft
}

func NewStepStreamDelete(streamService *service.Stream, draftService *service.StreamDraft, config datatype.Map) StepStreamDelete {
	return StepStreamDelete{
		streamService: streamService,
		draftService:  draftService,
	}
}

func (step StepStreamDelete) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step StepStreamDelete) Post(buffer io.Writer, renderer *Renderer) error {

	var parent model.Stream

	if err := step.streamService.LoadParent(renderer.stream, &parent); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamDelete.Post", "Error loading parent stream")
	}

	if err := step.streamService.Delete(renderer.stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamDelete.Post", "Error deleting stream")
	}

	if err := step.draftService.Delete(renderer.stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamDelete.Post", "Error deleting stream draft")
	}

	renderer.ctx.Response().Header().Set("HX-Redirect", "/"+parent.Token)
	return nil
}
