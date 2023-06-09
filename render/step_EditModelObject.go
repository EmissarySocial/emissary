package render

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
)

// StepEditModelObject is an action that can add new sub-streams to the domain.
type StepEditModelObject struct {
	Form    form.Element
	Options []*template.Template
}

// Get displays a modal form that lets users enter data for their new model object.
func (step StepEditModelObject) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepEditModelObject.Get"

	schema := renderer.schema()

	// Try to render the Form HTML
	result, err := form.Editor(schema, step.Form, renderer.object(), renderer.lookupProvider())

	if err != nil {
		return derp.Wrap(err, location, "Error generating form")
	}

	optionStrings := make([]string, len(step.Options))
	for index, option := range step.Options {
		optionStrings[index] = executeTemplate(option, renderer)
	}

	result = WrapForm(renderer.URL(), result, optionStrings...)

	// Wrap result as a modal dialog
	io.WriteString(buffer, result)

	return nil
}

func (step StepEditModelObject) UseGlobalWrapper() bool {
	return true
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepEditModelObject) Post(renderer Renderer, _ io.Writer) error {

	const location = "render.StepEditModelObject.Post"

	renderer.debug()

	// Get the request body
	body := mapof.NewAny()

	if err := renderer.context().Bind(&body); err != nil {
		return derp.Wrap(err, location, "Error binding request body")
	}

	spew.Dump(renderer.schema())

	// Appy request body to the object (limited and validated by the form schema)
	stepForm := form.New(renderer.schema(), step.Form)
	object := renderer.object()

	if err := stepForm.SetAll(object, body, renderer.lookupProvider()); err != nil {
		return derp.Wrap(err, location, "Error applying request body to model object", body)
	}

	// Save the object to the database
	if err := renderer.service().ObjectSave(object, "Edited"); err != nil {
		return derp.Wrap(err, location, "Error saving model object to database")
	}

	// Success!
	return nil
}
