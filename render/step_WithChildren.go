package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
)

// StepWithChildren represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithChildren struct {
	steps Pipeline

	BaseStep
}

// NewStepWithChildren returns a fully initialized StepWithChildren object
func NewStepWithChildren(stepInfo datatype.Map) (StepWithChildren, error) {

	const location = "NewStepWithChildren"

	steps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"))

	if err != nil {
		return StepWithChildren{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return StepWithChildren{
		steps: steps,
	}, nil
}

// Post updates the stream with approved data from the request body.
func (step StepWithChildren) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "render.StepWithChildren.Post"

	streamRenderer := renderer.(*Stream)

	children, err := factory.Stream().ListByParent(streamRenderer.stream.ParentID)

	if err != nil {
		return derp.Wrap(err, location, "Error listing children")
	}

	child := model.NewStream()

	for children.Next(&child) {

		// Make a renderer with the new child stream
		childStream, err := NewStreamWithoutTemplate(streamRenderer.factory(), streamRenderer.context(), &child, renderer.ActionID())

		if err != nil {
			return derp.Wrap(err, location, "Error creating renderer for child")
		}

		// Execute the POST render pipeline on the child
		if err := step.steps.Post(factory, &childStream, buffer); err != nil {
			return derp.Wrap(err, location, "Error executing steps for child")
		}

		// Reset the child object so that old records don't bleed into new ones.
		child = model.NewStream()
	}

	return nil
}
