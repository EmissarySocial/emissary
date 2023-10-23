package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithNextSibling represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithNextSibling struct {
	SubSteps []step.Step
}

func (step StepWithNextSibling) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodGet)
}

// Post executes the subSteps on the parent Stream
func (step StepWithNextSibling) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodPost)
}

// Post executes the subSteps on the parent Stream
func (step StepWithNextSibling) execute(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "render.StepWithNextSibling.Post"

	var sibling model.Stream

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer._stream

	if err := factory.Stream().LoadNextSibling(stream.ParentID, stream.Rank, &sibling); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error listing parent"))
	}

	// Make a renderer with the new parent stream
	// TODO: LOW: Is "view" really the best action to use here??
	siblingRenderer, err := NewStreamWithoutTemplate(streamRenderer.factory(), streamRenderer.request(), streamRenderer.response(), &sibling, "view")

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error creating renderer for sibling"))
	}

	// execute the POST render pipeline on the parent
	result := Pipeline(step.SubSteps).Execute(factory, &siblingRenderer, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for parent")

	return UseResult(result)
}
