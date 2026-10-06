package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithChildren is a Step that runs a sub-pipeline on every child of the current Stream
type StepWithChildren struct {
	SubSteps []step.Step
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepWithChildren) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post runs the sub-pipeline on each child, and stops at the first child that halts
func (step StepWithChildren) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepWithChildren.Post"

	factory := builder.factory()
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.Internal(location, "This step can only be used by Stream builders"))
	}

	// Walk this Stream's own children, not its siblings
	children, err := factory.Stream().RangeByParent(builder.session(), streamBuilder._stream.StreamID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Listing children"))
	}

	result := NewPipelineResult()

	for child := range children {

		// Make a builder with the new child stream
		// TODO: LOW: Is "view" really the best action to use here?
		childStream, err := NewStreamWithoutTemplate(streamBuilder.factory(), streamBuilder.session(), streamBuilder.request(), streamBuilder.response(), &child, "view")

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Creating builder for child"))
		}

		// Execute the POST build pipeline on the child
		childResult := Pipeline(step.SubSteps).Post(factory, &childStream, buffer)
		childResult.Error = derp.WrapIF(childResult.Error, location, "Executing steps for child")

		// Collect this child's results (including its Error and Halt) into the accumulator
		result.Merge(childResult)

		// RULE: Merge must happen BEFORE this check, or a child's Halt is only
		// noticed on the following iteration -- and never at all for the last child.
		// A failing step returns Halt().WithError(), so this covers errors too.
		if result.Halt {
			return UseResult(result)
		}
	}

	return UseResult(result)
}
