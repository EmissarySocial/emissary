package build

import (
	"io"
)

// StepRemoveEvent represents an action-step that forwards the user to a new page.
type StepRemoveEvent struct {
	Event string
}

func (step StepRemoveEvent) Get(_ Builder, _ io.Writer) PipelineBehavior {
	return Continue().RemoveEvent(step.Event)
}

// Post updates the stream with approved data from the request body.
func (step StepRemoveEvent) Post(_ Builder, _ io.Writer) PipelineBehavior {
	return Continue().RemoveEvent(step.Event)
}
