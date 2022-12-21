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

func (step StepSetHeader) Get(renderer Renderer, buffer io.Writer) error {
	if step.On == "get" || step.On == "both" {
		return step.setHeader(renderer)
	}
	return nil
}

func (step StepSetHeader) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepSetHeader) Post(renderer Renderer) error {
	if step.On == "post" || step.On == "both" {
		return step.setHeader(renderer)
	}
	return nil
}

func (step StepSetHeader) setHeader(renderer Renderer) error {

	response := renderer.context().Response()

	var value bytes.Buffer

	if err := step.Value.Execute(&value, renderer); err != nil {
		return derp.Wrap(err, "render.StepSetHeader.Post", "Error executing template", step.Value)
	}

	response.Header().Set(step.Name, value.String())

	return nil
}
