package build

import (
	"bytes"
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

// StepForwardTo is a Step that sends an HTMX 'forward' to a new page.
type StepForwardTo struct {
	URL    *template.Template
	Method string
}

// Get renders this step during a GET request. Implements the Step interface.
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
		return Halt().WithError(derp.Wrap(err, location, "Evaluating 'url'"))
	}

	// Reject dangerous or off-site-schemed targets. The value can be built from
	// remote-influenced data, so a `javascript:`/`data:` scheme (or a protocol-
	// relative host) must not become the forward target.
	if !uri.IsSafeRedirectURL(nextPage.String()) {
		return Halt().WithError(derp.BadRequest(location, "Unsafe forward target", nextPage.String()))
	}

	return Continue().WithEvent("closeModal", "true").WithHeader("Hx-Redirect", nextPage.String())
}
