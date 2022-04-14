package render

import (
	"bytes"
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
)

// StepLogActivity represents an action-step that can delete a Stream from the Domain
type StepLogActivity struct {
	Type    string
	Link    string
	Comment *template.Template
}

// Get displays a customizable confirmation form for the delete
func (step StepLogActivity) Get(renderer Renderer, buffer io.Writer) error {
	return step.logActivity(renderer)
}

func (step StepLogActivity) UseGlobalWrapper() bool {
	return true
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepLogActivity) Post(renderer Renderer, buffer io.Writer) error {
	return step.logActivity(renderer)
}

func (step StepLogActivity) logActivity(renderer Renderer) error {

	const location = "render.StepLogActivity.logActivity"

	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream

	// Only log activity when users are signed in
	if !streamRenderer.IsAuthenticated() {
		return nil
	}

	// Try to calculate the comment to be saved
	var comment bytes.Buffer

	if err := step.Comment.Execute(&comment, renderer); err != nil {
		return derp.Wrap(err, location, "Error generating comment", step)
	}

	// Create the new activity record
	activity := model.NewActivity()
	activity.StreamID = streamRenderer.objectID()
	activity.UserID = streamRenderer.UserID()
	activity.Type = step.Type
	activity.Comment = comment.String()

	switch step.Link {
	case "parent":
		activity.Link = streamRenderer.Host() + "/" + stream.ParentID.Hex()

	// case "self":
	default:
		activity.Link = streamRenderer.Host() + "/" + stream.StreamID.Hex()
	}

	// Try to save the new activity
	activityService := renderer.factory().Activity()

	if err := activityService.Save(&activity, ""); err != nil {
		return derp.Wrap(err, location, "Error saving activity", activity)
	}

	return nil
}
