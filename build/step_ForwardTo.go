package build

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/derp"
)

// StepForwardTo is an action-step that sends an HTMX 'forward' to a new page.
type StepForwardTo struct {
	URL *template.Template
}

func (step StepForwardTo) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepForwardTo) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepForwardTo.Post"
	var nextPage bytes.Buffer

	if err := step.URL.Execute(&nextPage, builder); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error evaluating 'url'"))
	}

	return Continue().WithEvent("closeModal", "true").WithHeader("Hx-Redirect", nextPage.String())
}
