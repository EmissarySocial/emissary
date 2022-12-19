package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
)

// StepForm represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepForm struct {
	Form    form.Element
	Target  string
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

	if step.Target == "" {
		step.Target = renderer.URL()
	}

	// Wrap result as a modal dialog and return to caller
	io.WriteString(buffer, WrapForm(step.Target, result, step.Options...))
	return nil
}

func (step StepForm) UseGlobalWrapper() bool {
	return true
}

// Post updates the object with approved data from the request body.
func (step StepForm) Post(renderer Renderer) error {

	const location = "render.StepForm.Post"

	request := renderer.context().Request()

	// Parse form information
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, location, "Error parsing form data")
	}

	object := renderer.object()
	form := form.New(renderer.schema(), step.Form)

	if err := form.SetAll(object, maps.FromURLValues(request.Form), renderer.factory().LookupProvider()); err != nil {
		return derp.Wrap(err, location, "Error setting form values")
	}

	return nil
}
