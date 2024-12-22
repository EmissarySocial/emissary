package build

import (
	"io"
)

// StepReloadPage is an action-step that forwards the user to a new page.
type StepReloadPage struct{}

func (step StepReloadPage) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepReloadPage) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue().WithHeader("HX-Refresh", "true")
}
