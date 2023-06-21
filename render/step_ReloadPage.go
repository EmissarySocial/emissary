package render

import (
	"io"
)

// StepReloadPage represents an action-step that forwards the user to a new page.
type StepReloadPage struct{}

func (step StepReloadPage) Get(renderer Renderer, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepReloadPage) Post(renderer Renderer, _ io.Writer) PipelineBehavior {
	return Continue().WithHeader("HX-Refresh", "true")
}
