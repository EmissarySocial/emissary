package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepDeleteAttachment represents an action that can upload attachments.  It can only be used on a StreamRenderer
type StepDeleteAttachment struct{}

func (step StepDeleteAttachment) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepDeleteAttachment) UseGlobalWrapper() bool {
	return true
}

func (step StepDeleteAttachment) Post(renderer Renderer) error {

	const location = "renderer.StepDeleteAttachment.Post"

	// TODO: could this be generalized to work with more than just streams???
	factory := renderer.factory()

	attachmentService := factory.Attachment()

	attachmentID := renderer.context().QueryParam("filename")

	attachment, err := attachmentService.LoadByToken(renderer.objectID(), attachmentID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading attachment")
	}

	// Delete the attachment
	if err := attachmentService.Delete(&attachment, "Deleted"); err != nil {
		return derp.Wrap(err, location, "Error deleting attachment")
	}

	// Notify the client
	renderer.context().Response().Header().Set("HX-Trigger", `attachments-updated`)

	return nil
}
