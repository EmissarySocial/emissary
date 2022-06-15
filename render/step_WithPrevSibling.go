package render

import (
	"bytes"
	"io"

	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/model/step"
)

// StepWithPrevSibling represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithPrevSibling struct {
	SubSteps []step.Step
}

func (step StepWithPrevSibling) Get(renderer Renderer, buffer io.Writer) error {
	return step.Execute(renderer, buffer, ActionMethodGet)
}

func (step StepWithPrevSibling) UseGlobalWrapper() bool {
	return useGlobalWrapper(step.SubSteps)
}

// Post executes the subSteps on the parent Stream
func (step StepWithPrevSibling) Post(renderer Renderer) error {
	var buffer bytes.Buffer
	return step.Execute(renderer, &buffer, ActionMethodPost)
}

// Post executes the subSteps on the parent Stream
func (step StepWithPrevSibling) Execute(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) error {

	const location = "render.StepWithPrevSibling.Post"

	var sibling model.Stream

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream

	if err := factory.Stream().LoadPrevSibling(stream.ParentID, stream.Rank, &sibling); err != nil {
		return derp.Wrap(err, location, "Error listing parent")
	}

	// Make a renderer with the new parent stream
	// TODO: Is "view" really the best action to use here??
	siblingRenderer, err := NewStreamWithoutTemplate(streamRenderer.factory(), streamRenderer.context(), &sibling, "view")

	if err != nil {
		return derp.Wrap(err, location, "Error creating renderer for sibling")
	}

	// Execute the POST render pipeline on the parent
	if err := Pipeline(step.SubSteps).Execute(factory, &siblingRenderer, buffer, actionMethod); err != nil {
		return derp.Wrap(err, location, "Error executing steps for parent")
	}

	return nil
}
