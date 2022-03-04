package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepForwardTo represents an action-step that forwards the user to a new page.
type StepForwardTo struct {
	url string
}

// NewStepForwardTo returns a fully initialized StepForwardTo object
func NewStepForwardTo(stepInfo datatype.Map) StepForwardTo {

	return StepForwardTo{
		url: stepInfo.GetString("url"),
	}
}

// Get displays a form where users can update stream data
func (step StepForwardTo) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepForwardTo) Post(buffer io.Writer, renderer Renderer) error {
	nextPage, err := executeSingleTemplate(step.url, renderer)

	if err != nil {
		return derp.Wrap(err, "render.StepForwardTo.Post", "Error executing template", step.url)
	}

	CloseModal(renderer.context(), nextPage)

	return nil
}
