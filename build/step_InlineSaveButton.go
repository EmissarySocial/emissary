package build

import (
	"io"
	"text/template"

	"github.com/benpate/html"
)

// StepInlineSaveButton is a Step that can build a Stream into HTML
type StepInlineSaveButton struct {
	ID    *template.Template
	Class string
	Label *template.Template
	Form  string
}

// Get builds the Stream HTML to the context
func (step StepInlineSaveButton) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepInlineSaveButton) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	h := html.New()

	id := executeTemplate(step.ID, builder)
	label := executeTemplate(step.Label, builder)

	// The replacement button must carry its own form association.  This step swaps out the
	// button the visitor clicked, so anything the original inherited from its surroundings is
	// lost -- a button inside its form inherits type=submit, but one in a menu bar needs an
	// explicit "form" to submit at all, and it is also what makes the SaveButton behavior's
	// "me.form" resolve.  Attr writes nothing when Form is empty.
	h.Button().
		ID(id).
		Attr("type", "submit").
		Attr("form", step.Form).
		Script("install SaveButton").
		Class(step.Class + " success").
		InnerHTML(label)

	if _, err := buffer.Write(h.Bytes()); err != nil {
		return Halt().WithError(err)
	}
	return Halt().WithHeader("HX-Reswap", "outerHTML").WithHeader("HX-Retarget", "#"+id)
}
