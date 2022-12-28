package render

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepEditModelObject is an action that can add new sub-streams to the domain.
type StepEditModelObject struct {
	Form    form.Element
	Options []*template.Template
}

// Get displays a modal form that lets users enter data for their new model object.
func (step StepEditModelObject) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepEditModelObject.Get"

	factory := renderer.factory()
	schema := renderer.schema()

	// Try to render the Form HTML
	result, err := form.Editor(schema, step.Form, renderer.object(), factory.LookupProvider())

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
func (step StepEditModelObject) Post(renderer Renderer) error {

	const location = "render.StepEditModelObject.Post"

	// This finds/creates a new object in the renderer
	request := renderer.context().Request()
	object := renderer.object()
	schema := renderer.schema()

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, location, "Error parsing form data")
	}

	// Try to set each path from the Form into the renderer.  Note: schema.Set also converts and validated inputs before setting.
	for key, value := range request.Form {
		if err := schema.Set(object, key, value); err != nil {
			return derp.Wrap(err, location, "Error setting path value", key, value)
		}
	}

	// Save the object to the database
	if err := renderer.service().ObjectSave(object, "Created"); err != nil {
		return derp.Wrap(err, location, "Error saving model object to database")
	}

	// Success!
	return nil
}
