package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepAddModelObject is an action that can add new model objects of any type
type StepAddModelObject struct {
	Form     form.Element
	Defaults []step.Step
}

// Get displays a modal form that lets users enter data for their new model object.
func (step StepAddModelObject) Get(renderer Renderer, buffer io.Writer) error {

	factory := renderer.factory()
	schema := renderer.schema()
	object := renderer.object()

	// First, try to execute any "default" steps so that the object is initialized
	if err := Pipeline(step.Defaults).Get(factory, renderer, buffer); err != nil {
		return derp.Wrap(err, "render.StepAddModelObject.Get", "Error executing default steps")
	}

	// Try to render the Form HTML
	result, err := form.Editor(schema, step.Form, object, factory.LookupProvider())

	if err != nil {
		return derp.Wrap(err, "render.StepAddModelObject.Get", "Error generating form")
	}

	result = WrapForm(renderer.URL(), result)

	// Wrap result as a modal dialog
	io.WriteString(buffer, result)
	return nil
}

func (step StepAddModelObject) UseGlobalWrapper() bool {
	return true
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepAddModelObject) Post(renderer Renderer) error {

	// This finds/creates a new object in the renderer
	factory := renderer.factory()
	request := renderer.context().Request()
	object := renderer.object()
	schema := renderer.schema()

	// Execute any "default" steps so that the object is initialized
	if err := Pipeline(step.Defaults).Post(factory, renderer); err != nil {
		return derp.Wrap(err, "render.StepAddModelObject.Get", "Error executing default steps")
	}

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, "render.AddModelObject.Post", "Error parsing form data")
	}

	// Try to set each path from the Form into the renderer.  Note: schema.Set also converts and validated inputs before setting.
	for key, value := range request.Form {
		if err := schema.Set(object, key, value); err != nil {
			return derp.Wrap(err, "render.AddModelObject.Post", "Error setting path value", key, value)
		}
	}

	// Save the object to the database
	if err := renderer.service().ObjectSave(object, "Created"); err != nil {
		return derp.Wrap(err, "render.StepAddModelObject.Post", "Error saving model object to database")
	}

	// Success!
	return nil
}
