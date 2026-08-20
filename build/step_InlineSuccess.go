package build

import (
	"io"
	"text/template"
)

// StepInlineSuccess is a Step that can build a Stream into HTML
type StepInlineSuccess struct {
	Message *template.Template
	Href    *template.Template
}

// Get builds the Stream HTML to the context
func (step StepInlineSuccess) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepInlineSuccess) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	result := executeTemplate(step.Message, builder)

	// If we have an HREF, then wrap the result in an anchor tag
	if href := executeTemplate(step.Href, builder); href != "" {
		result = `<a href="` + href + `" class="margin-left-sm">` + result + `</a>`
	} else {
		// Otherwise, write the result as green text.
		result = `<span class="text-green margin-left-sm">` + result + `</span>`
	}

	// Write to the buffer and return
	if _, err := buffer.Write([]byte(result)); err != nil {
		return Halt().WithError(err)
	}
	return Halt().WithHeader("HX-Reswap", "innerHTML").WithHeader("HX-Retarget", "#htmx-response-message")
}
