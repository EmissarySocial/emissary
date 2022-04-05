package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/model/step"
)

// StepWithParent represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithParent struct {
	SubSteps []step.Step
}

func (step StepWithParent) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

// Post executes the subSteps on the parent Stream
func (step StepWithParent) Post(renderer Renderer, buffer io.Writer) error {

	const location = "whisper.render.StepWithParent.Post"

	var parent model.Stream

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)

	if err := factory.Stream().LoadByID(streamRenderer.stream.ParentID, &parent); err != nil {
		return derp.Wrap(err, location, "Error listing parent")
	}

	// Make a renderer with the new parent stream
	parentStream, err := NewStreamWithoutTemplate(streamRenderer.factory(), streamRenderer.context(), &parent, renderer.ActionID())

	if err != nil {
		return derp.Wrap(err, location, "Error creating renderer for parent")
	}

	// Execute the POST render pipeline on the parent
	if err := Pipeline(step.SubSteps).Post(factory, &parentStream, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing steps for parent")
	}

	return nil
}
