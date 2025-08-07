package build

import (
	"io"
)

// StepAddEvent is a Step that forwards the user to a new page.
type StepAddEvent struct {
	Method string
	Event  string
}

func (step StepAddEvent) Get(_ Builder, _ io.Writer) PipelineBehavior {

	if step.Method == "post" {
		return nil
	}

	return Continue().WithEvent(step.Event, "true")
}

// Post updates the stream with approved data from the request body.
func (step StepAddEvent) Post(_ Builder, _ io.Writer) PipelineBehavior {

	if step.Method == "get" {
		return nil
	}

	return Continue().WithEvent(step.Event, "true")
}
