package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithDraft is a Step that runs its sub-steps against the Stream's draft
type StepWithDraft struct {
	SubSteps []step.Step
}

// Get runs the sub-steps' GET pipeline against the draft
func (step StepWithDraft) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepWithDraft.Get"

	factory := builder.factory()
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.Internal(location, "This step can only be used in a Stream builder"))
	}

	draftBuilder, err := streamBuilder.draftBuilder()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Getting draft builder"))
	}

	// Execute the GET build pipeline on the draft
	status := Pipeline(step.SubSteps).Get(factory, draftBuilder, buffer)
	status.Error = derp.WrapIF(status.Error, location, "Executing steps on draft")

	return UseResult(status)
}

// Post updates the stream with approved data from the request body.
func (step StepWithDraft) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepWithDraft.Post"

	factory := builder.factory()
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.Internal(location, "This step can only be used in a Stream builder"))
	}

	draftBuilder, err := streamBuilder.draftBuilder()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Getting draft builder"))
	}

	// Execute the POST build pipeline on the parent
	result := Pipeline(step.SubSteps).Post(factory, draftBuilder, buffer)
	result.Error = derp.WrapIF(result.Error, location, "Executing steps on draft")

	return UseResult(result)
}
