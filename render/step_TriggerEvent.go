package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepTriggerEvent represents an action-step that forwards the user to a new page.
type StepTriggerEvent struct {
	event string
	data  string
}

// NewStepTriggerEvent returns a fully initialized StepTriggerEvent object
func NewStepTriggerEvent(stepInfo datatype.Map) StepTriggerEvent {

	return StepTriggerEvent{
		event: stepInfo.GetString("event"),
		data:  stepInfo.GetString("data"),
	}
}

// Get displays a form where users can update stream data
func (step StepTriggerEvent) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepTriggerEvent) Post(buffer io.Writer, renderer Renderer) error {
	data, err := executeSingleTemplate(step.data, renderer)

	if err != nil {
		return derp.Wrap(err, "whisper.render.StepTriggerEvent.Post", "Error executing template", step.event, step.data)
	}

	if data == "" {
		renderer.context().Response().Header().Set("HX-Trigger", step.event)
	} else {
		renderer.context().Response().Header().Set("HX-Trigger", `{"`+step.event+`":`+data+`}`)
	}

	return nil
}
