package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/whisperverse/whisperverse/model/step"
)

// StepEditModelObject is an action that can add new sub-streams to the domain.
type StepEditModelObject struct {
	Form     form.Form
	Defaults []step.Step
}

// Get displays a modal form that lets users enter data for their new model object.
func (step StepEditModelObject) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepEditModelObject.Get"

	factory := renderer.factory()
	schema := renderer.schema()
	object := renderer.object()

	// First, try to execute any "default" steps so that the object is initialized
	if err := Pipeline(step.Defaults).Get(factory, renderer, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing default steps")
	}

	// Try to render the Form HTML
	result, err := step.Form.HTML(factory.FormLibrary(), &schema, object)

	if err != nil {
		return derp.Wrap(err, location, "Error generating form")
	}

	result = WrapForm(renderer.URL(), result)

	// Wrap result as a modal dialog
	io.WriteString(buffer, result)
	return nil
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepEditModelObject) Post(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepEditModelObject.Post"

	// This finds/creates a new object in the renderer
	factory := renderer.factory()
	request := renderer.context().Request()
	object := renderer.object()
	schema := renderer.schema()

	// Execute any "default" steps so that the object is initialized
	if err := Pipeline(step.Defaults).Post(factory, renderer, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing default steps")
	}

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, location, "Error parsing form data")
	}

	// Try to set each path from the Form into the renderer.  Note: schema.Set also converts and validated inputs before setting.
	for key, value := range request.Form {
		if err := schema.Set(renderer, key, value); err != nil {
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
