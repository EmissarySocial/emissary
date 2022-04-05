package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepForwardTo represents an action-step that forwards the user to a new page.
type StepForwardTo struct {
	url *template.Template

	BaseStep
}

// NewStepForwardTo returns a fully initialized StepForwardTo object
func NewStepForwardTo(stepInfo datatype.Map) (StepForwardTo, error) {

	const location = "render.NewStepForwardTo"

	url, err := template.New("").Parse(stepInfo.GetString("url"))

	if err != nil {
		return StepForwardTo{}, derp.Wrap(err, location, "Invalid 'url' template", stepInfo)
	}

	return StepForwardTo{
		url: url,
	}, nil
}

// Post updates the stream with approved data from the request body.
func (step StepForwardTo) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "render.StepForwardTo.Post"
	var nextPage bytes.Buffer

	if err := step.url.Execute(&nextPage, renderer); err != nil {
		return derp.Wrap(err, location, "Error evaluating 'url'")
	}

	CloseModal(renderer.context(), nextPage.String())

	return nil
}
