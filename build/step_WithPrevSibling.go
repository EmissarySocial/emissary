package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithPrevSibling is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithPrevSibling struct {
	SubSteps []step.Step
}

func (step StepWithPrevSibling) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post executes the subSteps on the parent Stream
func (step StepWithPrevSibling) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// Post executes the subSteps on the parent Stream
func (step StepWithPrevSibling) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithPrevSibling.execute"

	var sibling model.Stream

	factory := builder.factory()
	streamBuilder := builder.(Stream)
	stream := streamBuilder._stream

	if err := factory.Stream().LoadPrevSibling(builder.session(), stream.ParentID, stream.Rank, &sibling); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error listing parent"))
	}

	// Make a builder with the new parent stream
	// TODO: Is "view" really the best action to use here??
	siblingBuilder, err := NewStreamWithoutTemplate(streamBuilder.factory(), streamBuilder.session(), streamBuilder.request(), streamBuilder.response(), &sibling, "view")

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error creating builder for sibling"))
	}

	// Execute the POST build pipeline on the parent
	result := Pipeline(step.SubSteps).Execute(factory, &siblingBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for parent")
	return UseResult(result)
}
