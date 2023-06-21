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
func (step StepAsTooltip) Get(renderer Renderer, buffer io.Writer) ExitCondition {

	const location = "render.StepAsTooltip.Get"

	// Write inner items
	var tooltipBuffer bytes.Buffer

	status := Pipeline(step.SubSteps).Get(renderer.factory(), renderer, &tooltipBuffer)
	status.Error = derp.Wrap(status.Error, location, "Error executing subSteps")

	if status.Halt {
		return ExitWithStatus(status)
	}

	// Wrap the content in a tooltip
	tooltipContent := WrapTooltip(renderer.context().Response(), tooltipBuffer.String())

	if _, err := io.WriteString(buffer, tooltipContent); err != nil {
		return ExitError(derp.Wrap(err, location, "Error writing from builder to buffer"))
	}

	return ExitFullPage()
}

func (step StepAsTooltip) UseGlobalWrapper() bool {
	return false
}

// Post updates the stream with approved data from the request body.
func (step StepAsTooltip) Post(renderer Renderer, buffer io.Writer) ExitCondition {

	// Write inner items
	status := Pipeline(step.SubSteps).Post(renderer.factory(), renderer, buffer)
	status.Error = derp.Wrap(status.Error, "render.StepAsTooltip.Post", "Error executing subSteps")

	return ExitWithStatus(status).WithEvent("closeTooltip", "true")
}
