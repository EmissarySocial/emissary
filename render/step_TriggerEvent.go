package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepTriggerEvent represents an action-step that forwards the user to a new page.
type StepTriggerEvent struct {
	Event string
	Data  string
}

func (step StepTriggerEvent) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepTriggerEvent) Post(renderer Renderer, _ io.Writer) error {
	data, err := executeSingleTemplate(step.Data, renderer)

	if err != nil {
		return derp.Wrap(err, "render.StepTriggerEvent.Post", "Error executing template", step.Event, step.Data)
	}

	if data == "" {
		renderer.context().Response().Header().Set("HX-Trigger", step.Event)
	} else {
		renderer.context().Response().Header().Set("HX-Trigger", `{"`+step.Event+`":`+data+`}`)
	}

	return nil
}
