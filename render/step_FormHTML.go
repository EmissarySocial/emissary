package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepForm represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepForm struct {
	Form    form.Element
	Options []string
}

// Get displays a form where users can update stream data
func (step StepForm) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepForm.Get"

	// Try to render the Form HTML
	factory := renderer.factory()
	form := form.New(renderer.schema(), step.Form)

	result, err := form.Editor(renderer.object(), factory.LookupProvider())

	if err != nil {
		return derp.Wrap(err, location, "Error generating form")
	}

	// Wrap result as a modal dialog and return to caller
	io.WriteString(buffer, WrapForm(renderer.URL(), result, step.Options...))
	return nil
}

func (step StepForm) UseGlobalWrapper() bool {
	return true
}

// Post updates the object with approved data from the request body.
func (step StepForm) Post(renderer Renderer) error {

	const location = "render.StepForm.Post"

	request := renderer.context().Request()
	schema := renderer.schema()

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, location, "Error parsing form data")
	}

	object := renderer.object()

	// Try to set each path from the Form into the renderer.  Note: schema.Set also converts and validates inputs before setting.
	for _, element := range step.Form.AllElements() {
		value := request.Form[element.Path]
		if err := schema.Set(object, element.Path, value); err != nil {
			return derp.Wrap(err, location, "Error setting path value", element, value)
		}
	}

	if err := schema.Validate(object); err != nil {
		return derp.Wrap(err, location, "Object data is invalid")
	}

	return nil
}
