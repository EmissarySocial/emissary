package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithDraft represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithDraft struct {
	SubSteps []step.Step
}

// Get displays a form where users can update stream data
func (step StepWithDraft) Get(renderer Renderer, buffer io.Writer) ExitCondition {

	const location = "render.StepWithDraft.Get"

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)
	draftRenderer, err := streamRenderer.draftRenderer()

	if err != nil {
		return ExitError(derp.Wrap(err, location, "Error getting draft renderer"))
	}

	// Execute the POST render pipeline on the parent
	status := Pipeline(step.SubSteps).Get(factory, &draftRenderer, buffer)
	status.Error = derp.Wrap(status.Error, location, "Error executing steps on draft")

	return ExitWithStatus(status)
}

// Post updates the stream with approved data from the request body.
func (step StepWithDraft) Post(renderer Renderer, buffer io.Writer) ExitCondition {

	const location = "render.StepWithDraft.Post"

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)
	draftRenderer, err := streamRenderer.draftRenderer()

	if err != nil {
		return ExitError(derp.Wrap(err, location, "Error getting draft renderer"))
	}

	// Execute the POST render pipeline on the parent
	status := Pipeline(step.SubSteps).Post(factory, &draftRenderer, buffer)
	status.Error = derp.Wrap(status.Error, location, "Error executing steps on draft")

	return ExitWithStatus(status)
}
