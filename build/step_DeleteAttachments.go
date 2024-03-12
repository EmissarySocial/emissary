package build

import (
	"io"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepDeleteAttachments represents an action that can upload attachments.  It can only be used on a StreamBuilder
type StepDeleteAttachments struct {
	All bool
}

func (step StepDeleteAttachments) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

func (step StepDeleteAttachments) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "builder.StepDeleteAttachments.Post"

	factory := builder.factory()

	attachmentService := factory.Attachment()

	objectType := builder.service().ObjectType()
	objectID := builder.objectID()

	if step.All {

		// Delete all attachments for this stream
		if err := attachmentService.DeleteAll(objectType, objectID, "Deleted by {{.Author}}"); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error deleting all attachments"))
		}

	} else {

		attachmentIDString := builder.QueryParam("attachmentId")
		attachmentID, err := primitive.ObjectIDFromHex(attachmentIDString)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid attachment ID", attachmentIDString))
		}

		if err := attachmentService.DeleteByID(objectType, objectID, attachmentID, "Deleted by Workflow Step"); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error deleting attachment", attachmentID))
		}
	}

	// Notify the client
	builder.response().Header().Set("HX-Trigger", `attachments-updated`)

	return nil
}
