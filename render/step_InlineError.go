package render

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepInlineError represents an action-step that can render a Stream into HTML
type StepInlineError struct {
	Message *template.Template
}

// Get renders the Stream HTML to the context
func (step StepInlineError) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepInlineError) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {
	result := executeTemplate(step.Message, renderer)
	err := derp.NewInternalError("InlineError", result)

	if err := WrapInlineError(renderer.response(), err); err != nil {
		return Halt().WithError(err)
	}

	return Continue().AsFullPage()
}
