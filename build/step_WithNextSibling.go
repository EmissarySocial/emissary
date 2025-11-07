package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithNextSibling is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithNextSibling struct {
	SubSteps []step.Step
}

func (step StepWithNextSibling) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post executes the subSteps on the parent Stream
func (step StepWithNextSibling) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// Post executes the subSteps on the parent Stream
func (step StepWithNextSibling) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithNextSibling.Post"

	var sibling model.Stream

	factory := builder.factory()
	streamBuilder := builder.(Stream)
	stream := streamBuilder._stream

	if err := factory.Stream().LoadNextSibling(builder.session(), stream.ParentID, stream.Rank, &sibling); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error listing parent"))
	}

	// Make a builder with the new parent stream
	// TODO: LOW: Is "view" really the best action to use here??
	siblingBuilder, err := NewStreamWithoutTemplate(streamBuilder.factory(), builder.session(), streamBuilder.request(), streamBuilder.response(), &sibling, "view")

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create builder for sibling"))
	}

	// execute the POST build pipeline on the parent
	result := Pipeline(step.SubSteps).Execute(factory, &siblingBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Error executing steps for parent")

	return UseResult(result)
}
