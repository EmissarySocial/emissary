package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// InlineSaveButton is a Step that displays an "inline success" message on a form
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

// Name returns the name of the step, which is used in debugging.
func (step InlineSaveButton) Name() string {
	return "inline-save-button"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step InlineSaveButton) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step InlineSaveButton) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step InlineSaveButton) RequiredRoles() []string {
	return []string{}
}
