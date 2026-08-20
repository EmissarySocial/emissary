package build

import (
	"io"
	"text/template"
)

// StepTriggerEvent is a Step that forwards the user to a new page.
type StepTriggerEvent struct {
	Event string
	Value *template.Template
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepTriggerEvent) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepTriggerEvent) Post(builder Builder, _ io.Writer) PipelineBehavior {
	value := executeTemplate(step.Value, builder)
	return Continue().WithEvent(step.Event, value)
}
