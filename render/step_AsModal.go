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
	submit   string
}

// NewStepAsModal returns a fully initialized StepAsModal object
func NewStepAsModal(stepInfo datatype.Map) StepAsModal {

	return StepAsModal{
		subSteps: stepInfo.GetSliceOfMap("steps"),
		submit:   stepInfo.GetString("submit"),
	}
}

// Get displays a form where users can update stream data
func (step StepAsModal) Get(buffer io.Writer, renderer Renderer) error {

	header := renderer.context().Response().Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Push", "false")

	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal")
	b.Div().Class("modal-underlay").Script("on click send closeModal to #modal").Close()
	b.Div().Class("modal-content").EndBracket()

	// Write inner items
	if err := DoPipeline(renderer, b, step.subSteps, ActionMethodGet); err != nil {
		return derp.Wrap(err, "ghost.render.StepAsModal.Get", "Error executing subSteps")
	}

	// Done
	b.CloseAll()

	// Copy the modal dialog into the response buffer
	if _, err := buffer.Write([]byte(b.String())); err != nil {
		return derp.Wrap(err, "ghost.render.StepAsModal.Get", "Error writing from builder to buffer")
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepAsModal) Post(buffer io.Writer, renderer Renderer) error {

	// Write inner items
	if err := DoPipeline(renderer, buffer, step.subSteps, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepAsModal.Get", "Error executing subSteps")
	}

	closeModal(renderer.context(), "")
	return nil
}
