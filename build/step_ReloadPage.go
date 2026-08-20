package build

import (
	"io"
)

// StepReloadPage is a Step that forwards the user to a new page.
type StepReloadPage struct{}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepReloadPage) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepReloadPage) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue().WithHeader("HX-Refresh", "true")
}
