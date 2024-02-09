package render

import (
	"bytes"
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepSetHeader represents an action-step that can update the custom data stored in a Stream
type StepSetHeader struct {
	Method string
	Name   string
	Value  *template.Template
}

func (step StepSetHeader) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	if step.Method == "post" {
		return nil
	}
	return step.setHeader(renderer)
}

// Post updates the stream with approved data from the request body.
func (step StepSetHeader) Post(renderer Renderer, _ io.Writer) PipelineBehavior {
	if step.Method == "get" {
		return nil
	}
	return step.setHeader(renderer)
}

func (step StepSetHeader) setHeader(renderer Renderer) PipelineBehavior {

	var value bytes.Buffer

	if err := step.Value.Execute(&value, renderer); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepSetHeader.Post", "Error executing template", step.Value))
	}

	renderer.response().Header().Set(step.Name, value.String())

	return nil
}
