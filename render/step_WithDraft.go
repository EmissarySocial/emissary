package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model/step"
)

// StepWithDraft represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithDraft struct {
	SubSteps []step.Step
}

// Get displays a form where users can update stream data
func (step StepWithDraft) Get(renderer Renderer, buffer io.Writer) error {

	const location = "whisper.render.StepWithDraft.Get"

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)
	draftRenderer, err := streamRenderer.draftRenderer()

	if err != nil {
		return derp.Wrap(err, location, "Error getting draft renderer")
	}

	// Execute the POST render pipeline on the parent
	if err := Pipeline(step.SubSteps).Get(factory, &draftRenderer, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing steps on draft")
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepWithDraft) Post(renderer Renderer, buffer io.Writer) error {

	const location = "whisper.render.StepWithDraft.Post"

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)
	draftRenderer, err := streamRenderer.draftRenderer()

	if err != nil {
		return derp.Wrap(err, location, "Error getting draft renderer")
	}

	// Execute the POST render pipeline on the parent
	if err := Pipeline(step.SubSteps).Post(factory, &draftRenderer, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing steps for parent")
	}

	return nil
}
