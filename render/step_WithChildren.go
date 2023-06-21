package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithChildren represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithChildren struct {
	SubSteps []step.Step
}

func (step StepWithChildren) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepWithChildren) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {

	const location = "render.StepWithChildren.Post"

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)

	children, err := factory.Stream().ListByParent(streamRenderer.stream.ParentID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error listing children"))
	}

	child := model.NewStream()
	result := NewPipelineResult()

	for children.Next(&child) {

		// Make a renderer with the new child stream
		// TODO: LOW: Is "view" really the best action to use here??
		childStream, err := NewStreamWithoutTemplate(streamRenderer.factory(), streamRenderer.context(), &child, "")

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error creating renderer for child"))
		}

		// Execute the POST render pipeline on the child
		childResult := Pipeline(step.SubSteps).Post(factory, &childStream, buffer)
		childResult.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

		if result.Halt {
			return UseResult(result)
		}

		// Reset the child object so that old records don't bleed into new ones.
		child = model.NewStream()
		result.Merge(childResult)
	}

	return UseResult(result)
}
