package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
)

// StepWithDraft represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithDraft struct {
	streamService *service.Stream
	steps         []datatype.Map
}

// NewStepWithDraft returns a fully initialized StepWithDraft object
func NewStepWithDraft(streamService *service.Stream, stepInfo datatype.Map) StepWithDraft {

	return StepWithDraft{
		streamService: streamService,
		steps:         stepInfo.GetSliceOfMap("steps"),
	}
}

// Get displays a form where users can update stream data
func (step StepWithDraft) Get(buffer io.Writer, renderer Renderer) error {
	return step.execute(buffer, renderer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithDraft) Post(buffer io.Writer, renderer Renderer) error {
	return step.execute(buffer, renderer, ActionMethodPost)
}

// Execute makes a separate renderer bound to the StreamDraft service that executes a list of sub-steps on
// the draft copy of the provided Stream
func (step StepWithDraft) execute(buffer io.Writer, renderer Renderer, actionMethod ActionMethod) error {

	streamRenderer := renderer.(*Stream)
	draftRenderer := streamRenderer.draftRenderer()

	// Execute the POST render pipeline on the parent
	if err := DoPipeline(&draftRenderer, buffer, step.steps, actionMethod); err != nil {
		return derp.Wrap(err, "ghost.render.StepWithDraft.Post", "Error executing steps for parent")
	}

	return nil
}
