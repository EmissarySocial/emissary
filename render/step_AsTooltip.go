package render

import (
	"bytes"
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepAsTooltip represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepAsTooltip struct {
	SubSteps []step.Step
}

// Get displays a form where users can update stream data
func (step StepAsTooltip) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAsTooltip.Get"

	// Write inner items
	var tooltipBuffer bytes.Buffer

	// nolint:errcheck
	if err := Pipeline(step.SubSteps).Get(renderer.factory(), renderer, &tooltipBuffer); err != nil {
		return derp.Wrap(err, location, "Error executing subSteps")
	}

	// Wrap the content in a tooltip
	tooltipContent := WrapTooltip(renderer.context().Response(), tooltipBuffer.String())

	if _, err := io.WriteString(buffer, tooltipContent); err != nil {
		return derp.Wrap(err, location, "Error writing from builder to buffer")
	}

	return nil
}

func (step StepAsTooltip) UseGlobalWrapper() bool {
	return false
}

// Post updates the stream with approved data from the request body.
func (step StepAsTooltip) Post(renderer Renderer, buffer io.Writer) error {

	// Write inner items
	if err := Pipeline(step.SubSteps).Post(renderer.factory(), renderer, buffer); err != nil {
		return derp.Wrap(err, "render.StepAsTooltip.Post", "Error executing subSteps")
	}

	CloseTooltip(renderer.context())
	return nil
}
