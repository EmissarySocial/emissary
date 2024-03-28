package builder

import (
	"io"
	"text/template"

	"github.com/benpate/html"
)

// StepInlineSaveButton represents an action-step that can build a Stream into HTML
type StepInlineSaveButton struct {
	ID    *template.Template
	Class string
	Label *template.Template
}

// Get builds the Stream HTML to the context
func (step StepInlineSaveButton) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepInlineSaveButton) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	h := html.New()

	id := executeTemplate(step.ID, builder)
	label := executeTemplate(step.Label, builder)

	h.Button().ID(id).Script("install SaveButton").Class(step.Class + " success").InnerHTML(label)

	if _, err := buffer.Write(h.Bytes()); err != nil {
		return Halt().WithError(err)
	}
	return Halt().WithHeader("HX-Reswap", "outerHTML").WithHeader("HX-Retarget", "#"+id)
}
