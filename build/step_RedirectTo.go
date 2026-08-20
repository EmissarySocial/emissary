package build

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

// StepRedirectTo is a Step that sends an HTTP redirect to another page.
type StepRedirectTo struct {
	StatusCode int
	URL        *template.Template
	Method     string
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepRedirectTo) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	if step.Method != "post" {
		return step.execute(builder)
	}
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepRedirectTo) Post(builder Builder, _ io.Writer) PipelineBehavior {
	if step.Method != "get" {
		return step.execute(builder)
	}
	return nil
}

// Redirect returns an HTTP 307 Temporary Redirect that works for both GET and POST methods
func (step StepRedirectTo) execute(builder Builder) PipelineBehavior {

	const location = "build.StepRedirectTo.execute"
	var nextPage bytes.Buffer

	if err := step.URL.Execute(&nextPage, builder); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Evaluating 'url'"))
	}

	// Reject dangerous or off-site-schemed targets. The value can be built from
	// remote-influenced data, so a `javascript:`/`data:` scheme (or a protocol-
	// relative host) must not become the redirect target.
	if !uri.IsSafeRedirectURL(nextPage.String()) {
		return Halt().WithError(derp.BadRequest(location, "Unsafe redirect target", nextPage.String()))
	}

	if err := redirect(builder.response(), step.StatusCode, nextPage.String()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Redirecting to new page"))
	}

	return Halt().AsFullPage()
}
