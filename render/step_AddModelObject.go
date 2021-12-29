package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/path"
)

// StepAddModelObject is an action that can add new sub-streams to the domain.
type StepAddModelObject struct {
	formLibrary *form.Library
	form        form.Form
	defaults    []datatype.Map
}

// NewStepAddModelObject returns a fully initialized StepAddModelObject record
func NewStepAddModelObject(formLibrary *form.Library, stepInfo datatype.Map) StepAddModelObject {
	return StepAddModelObject{
		formLibrary: formLibrary,
		form:        form.MustParse(stepInfo.GetInterface("form")),
		defaults:    stepInfo.GetSliceOfMap("defaults"),
	}
}

// Get displays a modal form that lets users enter data for their new model object.
func (step StepAddModelObject) Get(buffer io.Writer, renderer Renderer) error {

	schema := renderer.schema()
	object := renderer.object()

	// First, try to execute any "default" steps so that the object is initialized
	if err := DoPipeline(renderer, buffer, step.defaults, ActionMethodGet); err != nil {
		return derp.Wrap(err, "ghost.render.StepAddModelObject.Get", "Error executing default steps")
	}

	// Try to render the Form HTML
	result, err := step.form.HTML(step.formLibrary, &schema, object)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepAddModelObject.Get", "Error generating form")
	}

	result = WrapForm(renderer, result)

	// Wrap result as a modal dialog
	io.WriteString(buffer, result)
	return nil
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepAddModelObject) Post(buffer io.Writer, renderer Renderer) error {

	// This finds/creates a new object in the renderer
	request := renderer.context().Request()
	object := renderer.object()
	schema := renderer.schema()

	// Execute any "default" steps so that the object is initialized
	if err := DoPipeline(renderer, buffer, step.defaults, ActionMethodGet); err != nil {
		return derp.Wrap(err, "ghost.render.StepAddModelObject.Get", "Error executing default steps")
	}

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, "ghost.render.AddModelObject.Post", "Error parsing form data")
	}

	// Try to set each path from the Form into the renderer.  Note: schema.Set also converts and validated inputs before setting.
	for key, value := range request.Form {
		if err := schema.Set(renderer, path.New(key), value); err != nil {
			return derp.Wrap(err, "ghost.render.AddModelObject.Post", "Error setting path value", key, value)
		}
	}

	// Save the object to the database
	if err := renderer.service().ObjectSave(object, "Created"); err != nil {
		return derp.Wrap(err, "ghost.render.StepAddModelObject.Post", "Error saving model object to database")
	}

	// Success!
	return nil
}
