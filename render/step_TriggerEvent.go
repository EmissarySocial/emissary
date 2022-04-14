package render

import (
	"bytes"
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepTriggerEvent represents an action-step that forwards the user to a new page.
type StepTriggerEvent struct {
	Event string
	Data  *template.Template
}

func (step StepTriggerEvent) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepTriggerEvent) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepTriggerEvent) Post(renderer Renderer, _ io.Writer) error {

	var buffer bytes.Buffer

	if err := step.Data.Execute(&buffer, renderer); err != nil {
		return derp.Wrap(err, "render.StepTriggerEvent.Post", "Error executing template", step.Event, step.Data)
	}

	data := buffer.String()

	if data == "" {
		renderer.context().Response().Header().Set("HX-Trigger", step.Event)
	} else {
		renderer.context().Response().Header().Set("HX-Trigger", `{"`+step.Event+`":`+data+`}`)
	}

	return nil
}
