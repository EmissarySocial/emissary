package render

import (
	"bytes"
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepSetHeader represents an action-step that can update the custom data stored in a Stream
type StepSetHeader struct {
	On    string
	Name  string
	Value *template.Template
}

func (step StepSetHeader) Get(renderer Renderer, buffer io.Writer) ExitCondition {
	if step.On == "post" {
		return nil
	}
	return step.setHeader(renderer)
}

// Post updates the stream with approved data from the request body.
func (step StepSetHeader) Post(renderer Renderer, _ io.Writer) ExitCondition {
	if step.On == "get" {
		return nil
	}
	return step.setHeader(renderer)
}

func (step StepSetHeader) setHeader(renderer Renderer) ExitCondition {

	response := renderer.context().Response()

	var value bytes.Buffer

	if err := step.Value.Execute(&value, renderer); err != nil {
		return ExitError(derp.Wrap(err, "render.StepSetHeader.Post", "Error executing template", step.Value))
	}

	response.Header().Set(step.Name, value.String())

	return nil
}
