package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithChildren is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithChildren struct {
	SubSteps []step.Step
}

func (step StepWithChildren) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepWithChildren) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepWithChildren.Post"

	factory := builder.factory()
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.InternalError(location, "This step can only be used by Stream builders"))
	}

	children, err := factory.Stream().RangeByParent(builder.session(), streamBuilder._stream.ParentID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to list children"))
	}

	result := NewPipelineResult()

	for child := range children {

		// Make a builder with the new child stream
		// TODO: LOW: Is "view" really the best action to use here?
		childStream, err := NewStreamWithoutTemplate(streamBuilder.factory(), streamBuilder.session(), streamBuilder.request(), streamBuilder.response(), &child, "view")

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Unable to create builder for child"))
		}

		// Execute the POST build pipeline on the child
		childResult := Pipeline(step.SubSteps).Post(factory, &childStream, buffer)
		childResult.Error = derp.WrapIF(result.Error, location, "Error executing steps for child")

		if result.Halt {
			return UseResult(result)
		}

		// Reset the child object so that old records don't bleed into new ones.
		result.Merge(childResult)
	}

	return UseResult(result)
}
