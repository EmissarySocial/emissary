package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/html"
)

// StepAsModal represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepAsModal struct {
	SubSteps []step.Step
	Class    string
}

// Get displays a form where users can update stream data
func (step StepAsModal) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAsModal.Get"

	header := renderer.context().Response().Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Push", "false")
	header.Set("HX-Reswap", "innerHTML")

	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal").Script("install Modal").Data("hx-swap", "none")
	b.Div().ID("modal-underlay").Close()
	b.Div().ID("modal-window").Class(step.Class).EndBracket()

	// Write inner items
	if err := Pipeline(step.SubSteps).Get(renderer.factory(), renderer, b); err != nil {
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

func (step StepAsModal) UseGlobalWrapper() bool {
	return false
}

// Post updates the stream with approved data from the request body.
func (step StepAsModal) Post(renderer Renderer) error {

	// Write inner items
	if err := Pipeline(step.SubSteps).Post(renderer.factory(), renderer); err != nil {
		return derp.Wrap(err, "render.StepAsModal.Post", "Error executing subSteps")
	}

	CloseModal(renderer.context(), "")
	return nil
}
