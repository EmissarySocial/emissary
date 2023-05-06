package render

import (
	"io"
)

// StepTriggerEvent represents an action-step that forwards the user to a new page.
type StepTriggerEvent struct {
	Event string
}

func (step StepTriggerEvent) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepTriggerEvent) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepTriggerEvent) Post(renderer Renderer, _ io.Writer) error {

	renderer.context().Response().Header().Set("HX-Trigger", step.Event)
	return nil
}
