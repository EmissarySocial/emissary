package render

import (
	"io"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	attachmentIDString := renderer.context().QueryParam("attachmentId")
	attachmentID, err := primitive.ObjectIDFromHex(attachmentIDString)

	if err != nil {
		return derp.Wrap(err, location, "Invalid attachment ID", attachmentIDString)
	}

	attachment, err := attachmentService.LoadByID(renderer.objectID(), attachmentID)

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
