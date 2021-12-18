package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StepWithParent represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithParent struct {
	streamService *service.Stream
	steps         []datatype.Map
}

// NewStepWithParent returns a fully initialized StepWithParent object
func NewStepWithParent(streamService *service.Stream, stepInfo datatype.Map) StepWithParent {

	return StepWithParent{
		streamService: streamService,
		steps:         stepInfo.GetSliceOfMap("steps"),
	}
}

// Get displays a form where users can update stream data
func (step StepWithParent) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepWithParent) Post(buffer io.Writer, renderer Renderer) error {

	var parent model.Stream

	streamRenderer := renderer.(*Stream)

	if err := step.streamService.LoadByID(streamRenderer.stream.ParentID, &parent); err != nil {
		return derp.Wrap(err, "ghost.render.StepWithParent.Post", "Error listing parent")
	}

	// Make a renderer with the new parent stream
	parentStream, err := NewStreamWithoutTemplate(streamRenderer.factory, streamRenderer.context(), &parent, renderer.ActionID())

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepWithParent.Post", "Error creating renderer for parent")
	}

	// Execute the POST render pipeline on the parent
	if err := DoPipeline(streamRenderer.factory, &parentStream, buffer, step.steps, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepWithParent.Post", "Error executing steps for parent")
	}

	return nil
}
