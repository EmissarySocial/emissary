package render

import (
	"io"
	"text/template"
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

	if _, err := buffer.Write([]byte(`<span class="red">` + result + `</span>`)); err != nil {
		return Halt().WithError(err)
	}

	return Halt().WithHeader("HX-Reswap", "innerHTML").WithHeader("HX-Retarget", "#htmx-response-message")
}
