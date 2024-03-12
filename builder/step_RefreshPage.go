package builder

import (
	"io"
)

// StepRefreshPage represents an action-step that forwards the user to a new page.
type StepRefreshPage struct{}

func (step StepRefreshPage) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepRefreshPage) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue().
		WithEvent("closeModal", "true").
		WithEvent("refreshPage", "true")
}
