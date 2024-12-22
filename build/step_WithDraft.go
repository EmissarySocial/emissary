package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithDraft is an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithDraft struct {
	SubSteps []step.Step
}

// Get displays a form where users can update stream data
func (step StepWithDraft) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepWithDraft.Get"

	factory := builder.factory()
	streamBuilder := builder.(*Stream)
	draftBuilder, err := streamBuilder.draftBuilder()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error getting draft builder"))
	}

	// Execute the POST build pipeline on the parent
	status := Pipeline(step.SubSteps).Get(factory, &draftBuilder, buffer)
	status.Error = derp.Wrap(status.Error, location, "Error executing steps on draft")

	return UseResult(status)
}

// Post updates the stream with approved data from the request body.
func (step StepWithDraft) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepWithDraft.Post"

	factory := builder.factory()
	streamBuilder := builder.(*Stream)
	draftBuilder, err := streamBuilder.draftBuilder()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error getting draft builder"))
	}

	// Execute the POST build pipeline on the parent
	result := Pipeline(step.SubSteps).Post(factory, &draftBuilder, buffer)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps on draft")

	return UseResult(result)
}
