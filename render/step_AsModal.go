package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/html"
)

// StepAsModal represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepAsModal struct {
	subSteps []datatype.Map
}

// NewStepAsModal returns a fully initialized StepAsModal object
func NewStepAsModal(stepInfo datatype.Map) StepAsModal {

	return StepAsModal{
		subSteps: stepInfo.GetSliceOfMap("steps"),
	}
}

// Get displays a form where users can update stream data
func (step StepAsModal) Get(buffer io.Writer, renderer Renderer) error {

	header := renderer.context().Response().Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Push", "false")

	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal").Script("install Modal").Data("hx-swap", "none")
	b.Div().Class("modal-underlay").Close()
	b.Div().Class("modal-content").EndBracket()

	// Write inner items
	if err := DoPipeline(renderer, b, step.subSteps, ActionMethodGet); err != nil {
		return derp.Wrap(err, "whisper.render.StepAsModal.Get", "Error executing subSteps")
	}

	// Done
	b.CloseAll()

	// Write the modal dialog into the response buffer
	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return derp.Wrap(err, "whisper.render.StepAsModal.Get", "Error writing from builder to buffer")
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepAsModal) Post(buffer io.Writer, renderer Renderer) error {

	// Write inner items
	if err := DoPipeline(renderer, buffer, step.subSteps, ActionMethodPost); err != nil {
		return derp.Wrap(err, "whisper.render.StepAsModal.Get", "Error executing subSteps")
	}

	CloseModal(renderer.context(), "")
	return nil
}
