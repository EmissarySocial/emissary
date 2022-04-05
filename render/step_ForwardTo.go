package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/derp"
)

// StepForwardTo represents an action-step that forwards the user to a new page.
type StepForwardTo struct {
	URL *template.Template
}

func (step StepForwardTo) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepForwardTo) Post(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepForwardTo.Post"
	var nextPage bytes.Buffer

	if err := step.URL.Execute(&nextPage, renderer); err != nil {
		return derp.Wrap(err, location, "Error evaluating 'url'")
	}

	CloseModal(renderer.context(), nextPage.String())

	return nil
}
