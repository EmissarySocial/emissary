package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
)

// EditModelObject is an action that can add new sub-streams to the domain.
type EditModelObject struct {
	Form    form.Element
	Options []*template.Template
}

// NewEditModelObject returns a fully initialized EditModelObject record
func NewEditModelObject(stepInfo maps.Map) (EditModelObject, error) {

	// Parse the form definition
	f, err := form.Parse(stepInfo.GetInterface("form"))

	if err != nil {
		return EditModelObject{}, derp.Wrap(err, "model.step.NewEditModelObject", "Invalid 'form'", stepInfo)
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
		Form:    f,
		Options: optionTemplates,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step EditModelObject) AmStep() {}
