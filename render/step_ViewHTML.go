package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepStreamHTML represents an action-step that can render a Stream into HTML
type StepStreamHTML struct {
	filename string
}

// NewStepStreamHTML generates a fully initialized StepStreamHTML step.
func NewStepStreamHTML(stepInfo datatype.Map) StepStreamHTML {

	filename := stepInfo.GetString("file")

	if filename == "" {
		filename = stepInfo.GetString("actionId")
	}

	return StepStreamHTML{
		filename: filename,
	}
}

// Get renders the Stream HTML to the context
func (step StepStreamHTML) Get(buffer io.Writer, renderer *Renderer) error {

	template, ok := renderer.template.HTMLTemplate(step.filename)

	if !ok {
		return derp.New(derp.CodeBadRequestError, "ghost.renderer.StepStreamHTML.Get", "Cannot find template", step.filename)
	}

	template.Funcs(FuncMap())

	if err := template.Execute(buffer, renderer); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamHTML.Get", "Error executing template")
	}

	return nil
}

// Post is not supported for this step.
func (step StepStreamHTML) Post(buffer io.Writer, renderer *Renderer) error {
	return derp.New(derp.CodeBadRequestError, "ghost.render.StepStreamHTML.Get", "Unsupported Method")
}
