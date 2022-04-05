package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepForm represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepForm struct {
	form form.Form

	BaseStep
}

// NewStepForm returns a fully initialized StepForm object
func NewStepForm(stepInfo datatype.Map) (StepForm, error) {

	f, err := form.Parse(stepInfo.GetInterface("form"))

	if err != nil {
		return StepForm{}, derp.Wrap(err, "render.NewStepForm", "Invalid 'form'", stepInfo)
	}

	return StepForm{
		form: f,
	}, nil
}

// Get displays a form where users can update stream data
func (step StepForm) Get(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "render.StepForm.Get"

	// Try to find the schema for this Template
	schema := renderer.schema()

	// Try to render the Form HTML
	result, err := step.form.HTML(factory.FormLibrary(), &schema, renderer.object())

	if err != nil {
		return derp.Wrap(err, location, "Error generating form")
	}

	// Wrap result as a modal dialog and return to caller
	io.WriteString(buffer, WrapForm(renderer.URL(), result))
	return nil
}

// Post updates the object with approved data from the request body.
func (step StepForm) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "render.StepForm.Post"

	request := renderer.context().Request()
	schema := renderer.schema()

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, location, "Error parsing form data")
	}

	object := renderer.object()

	// Try to set each path from the Form into the renderer.  Note: schema.Set also converts and validated inputs before setting.
	for _, element := range step.form.AllPaths() {
		value := request.Form[element.Path]
		if err := schema.Set(object, element.Path, value); err != nil {
			return derp.Wrap(err, location, "Error setting path value", element, value)
		}
	}

	return nil
}
