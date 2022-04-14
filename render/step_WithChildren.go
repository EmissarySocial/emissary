package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/model/step"
)

// StepWithChildren represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithChildren struct {
	SubSteps []step.Step
}

func (step StepWithChildren) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepWithChildren) UseGlobalWrapper() bool {
	return useGlobalWrapper(step.SubSteps)
}

// Post updates the stream with approved data from the request body.
func (step StepWithChildren) Post(renderer Renderer) error {

	const location = "render.StepWithChildren.Post"

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)

	children, err := factory.Stream().ListByParent(streamRenderer.stream.ParentID)

	if err != nil {
		return derp.Wrap(err, location, "Error listing children")
	}

	child := model.NewStream()

	for children.Next(&child) {

		// Make a renderer with the new child stream
		// TODO: Is "view" really the best action to use here??
		childStream, err := NewStreamWithoutTemplate(streamRenderer.factory(), streamRenderer.context(), &child, "")

		if err != nil {
			return derp.Wrap(err, location, "Error creating renderer for child")
		}

		// Execute the POST render pipeline on the child
		if err := Pipeline(step.SubSteps).Post(factory, &childStream); err != nil {
			return derp.Wrap(err, location, "Error executing steps for child")
		}

		// Reset the child object so that old records don't bleed into new ones.
		child = model.NewStream()
	}

	return nil
}
