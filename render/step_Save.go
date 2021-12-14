package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepStreamSave represents an action-step that can save changes to a Stream
type StepStreamSave struct {
	modelService ModelService
	comment      string
}

func NewStepStreamSave(modelService ModelService, stepInfo datatype.Map) StepStreamSave {

	return StepStreamSave{
		modelService: modelService,
		comment:      stepInfo.GetString("comment"),
	}
}

// Get displays a form for users to fill out in the browser
func (step StepStreamSave) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step StepStreamSave) Post(buffer io.Writer, renderer Renderer) error {

	// Try to update the stream
	if err := step.modelService.ObjectSave(renderer.object(), step.comment); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamSave.Post", "Error saving Stream")
	}

	return nil
}
