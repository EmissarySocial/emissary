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

// Post updates the object with approved data from the request body.
func (step StepForm) Post(buffer io.Writer, renderer Renderer) error {

	request := renderer.context().Request()
	schema := renderer.schema()

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepForm.Post", "Error parsing form data")
	}

	// Try to set each path from the Form into the renderer.  Note: schema.Set also converts and validated inputs before setting.
	for _, element := range step.form.AllPaths() {
		value := request.Form[element.Path]
		if err := schema.Set(renderer, path.New(element.Path), value); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepForm.Post", "Error setting path value", element, value)
		}
	}

	return nil
}
