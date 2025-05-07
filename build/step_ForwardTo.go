package build

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/derp"
)

// StepForwardTo is a Step that sends an HTMX 'forward' to a new page.
type StepForwardTo struct {
	URL    *template.Template
	Method string
}

func (step StepForwardTo) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if (step.Method == "get") || (step.Method == "both") {
		return step.do(builder)
	}

	return Continue()
}

// Post updates the stream with approved data from the request body.
func (step StepForwardTo) Post(builder Builder, _ io.Writer) PipelineBehavior {

	if step.Method == "post" || step.Method == "both" {
		return step.do(builder)
	}

	return Continue()
}

// Post updates the stream with approved data from the request body.
func (step StepForwardTo) do(builder Builder) PipelineBehavior {

	const location = "build.StepForwardTo.do"

	var nextPage bytes.Buffer

	if err := step.URL.Execute(&nextPage, builder); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error evaluating 'url'"))
	}

	return Continue().WithEvent("closeModal", "true").WithHeader("Hx-Redirect", nextPage.String())
}
