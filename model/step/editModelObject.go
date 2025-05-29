package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// EditModelObject is an action that can add new sub-streams to the domain.
type EditModelObject struct {
	Form    form.Element
	Options []*template.Template
}

// NewEditModelObject returns a fully initialized EditModelObject record
func NewEditModelObject(stepInfo mapof.Any) (EditModelObject, error) {

	formElement := form.NewElement()

	if formObject := stepInfo.GetAny("form"); formObject != nil {

		var err error

		// Parse the form definition
		formElement, err = form.Parse(stepInfo.GetAny("form"))

		if err != nil {
			return EditModelObject{}, derp.Wrap(err, "model.step.NewEditModelObject", "Invalid 'form'", stepInfo)
		}
	}

	// Parse options
	options := stepInfo.GetSliceOfString("options")
	optionTemplates := make([]*template.Template, len(options))

	for index, option := range options {
		template, err := template.New("option").Parse(option)

		if err != nil {
			return EditModelObject{}, derp.Wrap(err, "model.step.NewEditModelObject", "Invalid 'options'", stepInfo)
		}

		optionTemplates[index] = template
	}

	return EditModelObject{
		Form:    formElement,
		Options: optionTemplates,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step EditModelObject) Name() string {
	return "edit"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step EditModelObject) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step EditModelObject) RequiredRoles() []string {
	return []string{}
}
