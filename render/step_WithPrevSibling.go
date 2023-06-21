package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithPrevSibling represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithPrevSibling struct {
	SubSteps []step.Step
}

func (step StepWithPrevSibling) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodGet)
}

// Post executes the subSteps on the parent Stream
func (step StepWithPrevSibling) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodPost)
}

// Post executes the subSteps on the parent Stream
func (step StepWithPrevSibling) execute(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "render.StepWithPrevSibling.execute"

	var sibling model.Stream

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream

	if err := factory.Stream().LoadPrevSibling(stream.ParentID, stream.Rank, &sibling); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error listing parent"))
	}

	// Make a renderer with the new parent stream
	// TODO: Is "view" really the best action to use here??
	siblingRenderer, err := NewStreamWithoutTemplate(streamRenderer.factory(), streamRenderer.context(), &sibling, "view")

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error creating renderer for sibling"))
	}

	// Execute the POST render pipeline on the parent
	result := Pipeline(step.SubSteps).Execute(factory, &siblingRenderer, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for parent")
	return UseResult(result)
}
