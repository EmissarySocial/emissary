package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepServerRedirect represents an action-step that continues rendering the output stream as
// a GET request to a new action.
type StepServerRedirect struct {
	On     string // "get" or "post" or "both"
	Action string
}

func (step StepServerRedirect) Get(renderer Renderer, buffer io.Writer) ExitCondition {

	if step.On == "post" {
		return nil
	}

	return step.redirect(renderer, buffer)
}

// Post updates the stream with approved data from the request body.
func (step StepServerRedirect) Post(renderer Renderer, _ io.Writer) ExitCondition {
	if step.On == "get" {
		return nil
	}

	return step.redirect(renderer, renderer.context().Response())
}

// redirect creates a new renderer on this object with the requested Action and then continues as a GET request.
func (step StepServerRedirect) redirect(renderer Renderer, buffer io.Writer) ExitCondition {

	newRenderer, err := renderer.clone(step.Action)

	if err != nil {
		return ExitError(derp.Wrap(err, "render.StepServerRedirect.Redirect", "Error creating new renderer"))
	}

	result, err := newRenderer.Render()

	if err != nil {
		return ExitError(derp.Wrap(err, "render.StepServerRedirect.Redirect", "Error rendering new page"))
	}

	if _, err := buffer.Write([]byte(result)); err != nil {
		return ExitError(derp.Wrap(err, "render.StepServerRedirect.Redirect", "Error writing output buffer"))
	}

	return nil
}
