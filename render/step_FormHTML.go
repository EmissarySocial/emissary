package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/path"
)

// StepForm represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepForm struct {
	formLibrary form.Library
	form        form.Form
}

// NewStepForm returns a fully initialized StepForm object
func NewStepForm(formLibrary form.Library, stepInfo datatype.Map) StepForm {

	return StepForm{
		formLibrary: formLibrary,
		form:        form.MustParse(stepInfo.GetInterface("form")),
	}
}

// Get displays a form where users can update stream data
func (step StepForm) Get(buffer io.Writer, renderer Renderer) error {

	// Try to find the schema for this Template
	schema := renderer.schema()

	// Try to render the Form HTML
	result, err := step.form.HTML(step.formLibrary, &schema, renderer.object())

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepForm.Get", "Error generating form")
	}

	result = WrapForm(renderer, result)

	// Wrap result as a modal dialog
	buffer.Write([]byte(result))
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepForm) Post(buffer io.Writer, renderer Renderer) error {

	schema := renderer.schema()
	inputs := make(datatype.Map)

	// Collect form POST information
	if err := renderer.context().Bind(&inputs); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepForm.Post", "Error binding body")
	}

	if err := schema.Validate(inputs); err != nil {
		return derp.Wrap(err, "ghost.render.StepForm.Post", "Error validating input", inputs)
	}

	// Put approved form data into the stream
	if err := path.SetAll(renderer, inputs); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepStreamData.Post", "Error seting value", inputs)
	}

	return nil
}
