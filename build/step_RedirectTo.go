package build

import (
	"bytes"
	"html/template"
	"io"
	"net/http"

	"github.com/benpate/derp"
)

// StepRedirectTo represents an action-step that sends an HTTP redirect to another page.
type StepRedirectTo struct {
	URL *template.Template
}

func (step StepRedirectTo) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder)
}

// Post updates the stream with approved data from the request body.
func (step StepRedirectTo) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return step.execute(builder)
}

// Redirect returns an HTTP 307 Temporary Redirect that works for both GET and POST methods
func (step StepRedirectTo) execute(builder Builder) PipelineBehavior {

	const location = "build.StepRedirectTo.execute"
	var nextPage bytes.Buffer

	if err := step.URL.Execute(&nextPage, builder); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error evaluating 'url'"))
	}

	if err := redirect(builder.response(), http.StatusTemporaryRedirect, nextPage.String()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error redirecting to new page"))
	}

	return nil
}
