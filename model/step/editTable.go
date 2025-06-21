package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// TableEditor is an action that can add new sub-streams to the domain.
type TableEditor struct {
	Path string
	Form form.Element
}

// NewTableEditor returns a fully initialized TableEditor record
func NewTableEditor(stepInfo mapof.Any) (TableEditor, error) {

	f, err := form.Parse(stepInfo.GetAny("form"))

	if err != nil {
		return TableEditor{}, derp.Wrap(err, "model.step.NewTableEditor", "Invalid 'form'", stepInfo)
	}

	return TableEditor{
		Path: stepInfo.GetString("path"),
		Form: f,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step TableEditor) Name() string {
	return "edit-table"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step TableEditor) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step TableEditor) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step TableEditor) RequiredRoles() []string {
	return []string{}
}
