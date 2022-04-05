package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/html"
)

// StepAsModal represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepAsModal struct {
	subSteps Pipeline

	BaseStep
}

// NewStepAsModal returns a fully initialized StepAsModal object
func NewStepAsModal(stepInfo datatype.Map) (StepAsModal, error) {

	subSteps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"))

	if err != nil {
		return StepAsModal{}, derp.Wrap(err, "render.NewStepAsModal", "Invalid 'steps'", stepInfo)
	}

	return StepAsModal{
		subSteps: subSteps,
	}, nil
}

// Get displays a form where users can update stream data
func (step StepAsModal) Get(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAsModal.Get"

	header := renderer.context().Response().Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Push", "false")

	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal").Script("install Modal").Data("hx-swap", "none")
	b.Div().Class("modal-underlay").Close()
	b.Div().Class("modal-content").EndBracket()

	// Write inner items
	if err := step.subSteps.Get(factory, renderer, b); err != nil {
		return derp.Wrap(err, location, "Error executing subSteps")
	}

	// Done
	b.CloseAll()

	// Write the modal dialog into the response buffer
	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return derp.Wrap(err, location, "Error writing from builder to buffer")
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepAsModal) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	// Write inner items
	if err := step.subSteps.Post(factory, renderer, buffer); err != nil {
		return derp.Wrap(err, "render.StepAsModal.Post", "Error executing subSteps")
	}

	CloseModal(renderer.context(), "")
	return nil
}
