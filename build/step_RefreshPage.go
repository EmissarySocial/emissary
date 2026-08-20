package build

import (
	"io"
)

// StepRefreshPage is a Step that forwards the user to a new page.
type StepRefreshPage struct{}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepRefreshPage) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepRefreshPage) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue().
		WithEvent("closeModal", "true").
		WithEvent("refreshPage", "true")
}
