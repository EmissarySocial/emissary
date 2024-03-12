package builder

import (
	"io"
	"text/template"
)

// StepInlineError represents an action-step that can build a Stream into HTML
type StepInlineError struct {
	Message *template.Template
}

// Get builds the Stream HTML to the context
func (step StepInlineError) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepInlineError) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	result := executeTemplate(step.Message, builder)

	if _, err := buffer.Write([]byte(`<span class="red">` + result + `</span>`)); err != nil {
		return Halt().WithError(err)
	}

	return Halt().WithHeader("HX-Reswap", "innerHTML").WithHeader("HX-Retarget", "#htmx-response-message")
}
