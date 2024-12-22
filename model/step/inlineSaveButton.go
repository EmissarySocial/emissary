package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// InlineSaveButton is an action-step that displays an "inline success" message on a form
type InlineSaveButton struct {
	ID    *template.Template
	Class string
	Label *template.Template
}

func NewInlineSaveButton(stepInfo mapof.Any) (InlineSaveButton, error) {

	// Get the ID.  Default is "inline-save-button"
	id := first(stepInfo.GetString("id"), "inline-save-button")
	idTemplate, err := template.New("").Funcs(FuncMap()).Parse(id)

	if err != nil {
		return InlineSaveButton{}, derp.Wrap(err, "model.step.NewInlineSaveButton", "Error parsing template")
	}

	// Get the Label.  Default is "Save Changes"
	label := first(stepInfo.GetString("label"), "Save Changes")
	labelTemplate, err := template.New("").Funcs(FuncMap()).Parse(label)

	if err != nil {
		return InlineSaveButton{}, derp.Wrap(err, "model.step.NewInlineSaveButton", "Error parsing template")
	}

	return InlineSaveButton{
		ID:    idTemplate,
		Class: first(stepInfo.GetString("class"), "primary"),
		Label: labelTemplate,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step InlineSaveButton) AmStep() {}
