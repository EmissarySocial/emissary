package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepForm represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepForm struct {
	Form form.Form
}

// Get displays a form where users can update stream data
func (step StepForm) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepForm.Get"

	// Try to find the schema for this Template
	factory := renderer.factory()
	schema := renderer.schema()

	// Try to render the Form HTML
	result, err := step.Form.HTML(factory.FormLibrary(), &schema, renderer.object())

	if err != nil {
		return derp.Wrap(err, location, "Error generating form")
	}

	// Wrap result as a modal dialog and return to caller
	io.WriteString(buffer, WrapForm(renderer.URL(), result))
	return nil
}

// Post updates the object with approved data from the request body.
func (step StepForm) Post(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepForm.Post"

	request := renderer.context().Request()
	schema := renderer.schema()

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, location, "Error parsing form data")
	}

	object := renderer.object()

	// Try to set each path from the Form into the renderer.  Note: schema.Set also converts and validated inputs before setting.
	for _, element := range step.Form.AllPaths() {
		value := request.Form[element.Path]
		if err := schema.Set(object, element.Path, value); err != nil {
			return derp.Wrap(err, location, "Error setting path value", element, value)
		}
	}

	return nil
}
