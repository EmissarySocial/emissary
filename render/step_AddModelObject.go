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
func (step StepAddModelObject) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {

	factory := renderer.factory()
	schema := renderer.schema()
	object := renderer.object()

	// First, try to execute any "default" steps so that the object is initialized
	result := Pipeline(step.Defaults).Get(factory, renderer, buffer)

	if result.Halt {
		result.Error = derp.Wrap(result.Error, "render.StepAddModelObject.Get", "Error executing default steps")
		return UseResult(result)
	}

	// Try to render the Form HTML
	formHTML, err := form.Editor(schema, step.Form, object, renderer.lookupProvider())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepAddModelObject.Get", "Error generating form"))
	}

	formHTML = WrapForm(renderer.URL(), formHTML)

	// Wrap formHTML as a modal dialog
	// nolint:errcheck
	io.WriteString(buffer, formHTML)
	return nil
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepAddModelObject) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {

	// This finds/creates a new object in the renderer
	factory := renderer.factory()
	request := renderer.request()
	object := renderer.object()
	schema := renderer.schema()

	// Execute any "default" steps so that the object is initialized
	result := Pipeline(step.Defaults).Post(factory, renderer, buffer)

	if result.Halt {
		result.Error = derp.Wrap(result.Error, "render.StepAddModelObject.Post", "Error executing default steps")
		return UseResult(result)
	}

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.AddModelObject.Post", "Error parsing form data"))
	}

	// Try to set each path from the Form into the renderer.  Note: schema.Set also converts and validated inputs before setting.
	for key, value := range request.Form {
		if err := schema.Set(object, key, value); err != nil {
			return Halt().WithError(derp.Wrap(err, "render.AddModelObject.Post", "Error setting path value", key, value))
		}
	}

	// Save the object to the database
	if err := renderer.service().ObjectSave(object, "Created"); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepAddModelObject.Post", "Error saving model object to database"))
	}

	// Success!
	return nil
}
