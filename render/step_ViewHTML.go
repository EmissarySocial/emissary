package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepViewHTML represents an action-step that can render a Stream into HTML
type StepViewHTML struct {
	filename string
}

// NewStepViewHTML generates a fully initialized StepViewHTML step.
func NewStepViewHTML(stepInfo datatype.Map) StepViewHTML {

	filename := stepInfo.GetString("file")

	if filename == "" {
		filename = stepInfo.GetString("actionId")
	}

	return StepViewHTML{
		filename: filename,
	}
}

// Get renders the Stream HTML to the context
func (step StepViewHTML) Get(buffer io.Writer, renderer Renderer) error {

	if err := renderer.executeTemplate(buffer, step.filename, renderer); err != nil {
		return derp.Wrap(err, "whisper.render.StepViewHTML.Get", "Error executing template")
	}

	return nil
}

// Post is not supported for this step.
func (step StepViewHTML) Post(buffer io.Writer, renderer Renderer) error {
	return nil
}
