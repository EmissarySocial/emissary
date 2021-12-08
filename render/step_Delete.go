package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
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

	if err := step.streamService.Delete(renderer.stream, "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamDelete.Post", "Error deleting stream")
	}

	return nil
}
