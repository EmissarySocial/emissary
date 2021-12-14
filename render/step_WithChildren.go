package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StepWithChildren represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithChildren struct {
	streamService *service.Stream
	steps         []datatype.Map
}

// NewStepWithChildren returns a fully initialized StepWithChildren object
func NewStepWithChildren(streamService *service.Stream, stepInfo datatype.Map) StepWithChildren {

	return StepWithChildren{
		streamService: streamService,
		steps:         stepInfo.GetSliceOfMap("steps"),
	}
}

// Get displays a form where users can update stream data
func (step StepWithChildren) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepWithChildren) Post(buffer io.Writer, renderer Renderer) error {

	streamRenderer := renderer.(Stream)

	children, err := step.streamService.ListByParent(streamRenderer.stream.ParentID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepWithChildren.Post", "Error listing children")
	}

	child := new(model.Stream)

	for children.Next(child) {

		// Make a renderer with the new child stream
		childStream, err := NewStreamWithoutTemplate(streamRenderer.factory, streamRenderer.context(), *child, renderer.ActionID())

		if err != nil {
			return derp.Wrap(err, "ghost.render.StepWithChildren.Post", "Error creating renderer for child")
		}

		// Execute the POST render pipeline on the child
		if err := DoPipeline(streamRenderer.factory, &childStream, buffer, step.steps, ActionMethodPost); err != nil {
			return derp.Wrap(err, "ghost.render.StepWithChildren.Post", "Error executing steps for child")
		}

		// Reset the child object so that old records don't bleed into new ones.
		child = new(model.Stream)
	}

	return nil
}
