package build

import (
	"io"
	"text/template"
)

// StepInlineSuccess is an action-step that can build a Stream into HTML
type StepInlineSuccess struct {
	Message *template.Template
}

// Get builds the Stream HTML to the context
func (step StepInlineSuccess) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepInlineSuccess) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	result := executeTemplate(step.Message, builder)

	if _, err := buffer.Write([]byte(`<span class="green">` + result + `</span>`)); err != nil {
		return Halt().WithError(err)
	}
	return Halt().WithHeader("HX-Reswap", "innerHTML").WithHeader("HX-Retarget", "#htmx-response-message")
}
