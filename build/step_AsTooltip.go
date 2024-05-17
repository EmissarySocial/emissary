package build

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
func (step StepAsTooltip) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepAsTooltip.Get"

	// Write inner items
	var tooltipBuffer bytes.Buffer

	result := Pipeline(step.SubSteps).Get(builder.factory(), builder, &tooltipBuffer)
	result.Error = derp.Wrap(result.Error, location, "Error executing subSteps")

	if result.Halt {
		return UseResult(result)
	}

	// Wrap the content in a tooltip
	tooltipContent := WrapTooltip(builder.response(), tooltipBuffer.String())

	if _, err := io.WriteString(buffer, tooltipContent); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error writing from builder to buffer"))
	}

	return Halt().AsFullPage()
}

func (step StepAsTooltip) UseGlobalWrapper() bool {
	return false
}

// Post updates the stream with approved data from the request body.
func (step StepAsTooltip) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	// Write inner items
	result := Pipeline(step.SubSteps).Post(builder.factory(), builder, buffer)
	result.Error = derp.Wrap(result.Error, "build.StepAsTooltip.Post", "Error executing subSteps")

	return UseResult(result).WithEvent("closeTooltip", "true")
}
