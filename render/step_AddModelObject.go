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
	modelService ModelService
	formLibrary  form.Library
	form         form.Form
	defaults     []datatype.Map
}

// NewStepAddModelObject returns a fully initialized StepAddModelObject record
func NewStepAddModelObject(modelService ModelService, formLibrary form.Library, stepInfo datatype.Map) StepAddModelObject {
	return StepAddModelObject{
		modelService: modelService,
		formLibrary:  formLibrary,
		form:         form.MustParse(stepInfo.GetInterface("form")),
		defaults:     stepInfo.GetSliceOfMap("defaults"),
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
	buffer.Write([]byte(result))
	return nil
}

func (step StepAddModelObject) Post(buffer io.Writer, renderer Renderer) error {

	// This finds/creates a new object in the renderer
	object := renderer.object()
	schema := renderer.schema()
	inputs := make(datatype.Map)

	// Execute any "default" steps so that the object is initialized
	if err := DoPipeline(renderer, buffer, step.defaults, ActionMethodGet); err != nil {
		return derp.Wrap(err, "ghost.render.StepAddModelObject.Get", "Error executing default steps")
	}

	// Collect form POST information
	if err := renderer.context().Bind(&inputs); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepAddModelObject.Post", "Error binding body")
	}

	// Validate form inputs
	if err := schema.Validate(inputs); err != nil {
		return derp.Wrap(err, "ghost.render.StepAddModelObject.Post", "Error validating input", inputs)
	}

	// Set form values into object
	if err := path.SetAll(renderer, inputs); err != nil {
		return derp.Wrap(err, "ghost.render.StepAddModelObject.Post", "Error seting values", inputs)
	}

	// Save the object to the database
	if err := step.modelService.ObjectSave(object, "Created"); err != nil {
		return derp.Wrap(err, "ghost.render.StepAddModelObject.Post", "Error saving model object to database")
	}

	// Close modal dialog
	// TODO: this should move somewhere else later..
	closeModal(renderer.context(), "")

	return nil
}
