package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
)

// StepWithParent represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithParent struct {
	subSteps Pipeline

	BaseStep
}

// NewStepWithParent returns a fully initialized StepWithParent object
func NewStepWithParent(stepInfo datatype.Map) (StepWithParent, error) {

	const location = "render.NewStepWithParent"

	subSteps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"))

	if err != nil {
		return StepWithParent{}, derp.Wrap(err, location, "Invalid 'steps'")
	}

	return StepWithParent{
		subSteps: subSteps,
	}, nil
}

// Post executes the subSteps on the parent Stream
func (step StepWithParent) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "whisper.render.StepWithParent.Post"

	var parent model.Stream

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
	if err := step.subSteps.Post(factory, &parentStream, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing steps for parent")
	}

	return nil
}
