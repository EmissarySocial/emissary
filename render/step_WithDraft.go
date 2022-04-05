package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepWithDraft represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithDraft struct {
	subSteps Pipeline

	BaseStep
}

// NewStepWithDraft returns a fully initialized StepWithDraft object
func NewStepWithDraft(stepInfo datatype.Map) (StepWithDraft, error) {

	const location = "render.NewStepWithDraft"

	subSteps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"))

	if err != nil {
		return StepWithDraft{}, derp.Wrap(err, location, "Invalid 'steps'")
	}

	return StepWithDraft{
		subSteps: subSteps,
	}, nil
}

// Get displays a form where users can update stream data
func (step StepWithDraft) Get(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "whisper.render.StepWithDraft.Get"

	streamRenderer := renderer.(*Stream)
	draftRenderer, err := streamRenderer.draftRenderer()

	if err != nil {
		return derp.Wrap(err, location, "Error getting draft renderer")
	}

	// Execute the POST render pipeline on the parent
	if err := step.subSteps.Get(factory, &draftRenderer, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing steps on draft")
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepWithDraft) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "whisper.render.StepWithDraft.Post"

	streamRenderer := renderer.(*Stream)
	draftRenderer, err := streamRenderer.draftRenderer()

	if err != nil {
		return derp.Wrap(err, location, "Error getting draft renderer")
	}

	// Execute the POST render pipeline on the parent
	if err := step.subSteps.Post(factory, &draftRenderer, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing steps for parent")
	}

	return nil
}
