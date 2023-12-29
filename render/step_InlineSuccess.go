package render

import (
	"io"
	"text/template"
)

// StepInlineSuccess represents an action-step that can render a Stream into HTML
type StepInlineSuccess struct {
	Message *template.Template
}

// Get renders the Stream HTML to the context
func (step StepInlineSuccess) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepInlineSuccess) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {
	result := executeTemplate(step.Message, renderer)

	if err := WrapInlineSuccess(renderer.response(), result); err != nil {
		return Halt().WithError(err)
	}

	return Continue().AsFullPage()
}
