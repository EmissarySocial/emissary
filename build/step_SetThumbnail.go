package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepSetThumbnail is a Step that can update the data.DataMap custom data stored in a Stream
type StepSetThumbnail struct {
	Path string
}

func (step StepSetThumbnail) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSetThumbnail) Post(builder Builder, _ io.Writer) PipelineBehavior {

	// Find best icon from attachments
	factory := builder.factory()

	objectType := builder.service().ObjectType()
	objectID := builder.objectID()
	object := builder.object()

	attachments, err := factory.Attachment().QueryByObjectID(builder.session(), objectType, objectID)

	if err != nil {
		return Halt().WithError(derp.BadRequestError("build.StepSetThumbnail.Post", "Unable to list attachments"))
	}

	// Scan all attachments and use the first one that is an image.

	schema := builder.schema()

	for _, attachment := range attachments {
		if attachment.MimeCategory() == "image" {

			// Special case for User objects (this should always be "iconId")
			if objectType == "User" {
				if err := schema.Set(object, step.Path, attachment.AttachmentID.Hex()); err != nil {
					return Halt().WithError(derp.InternalError("build.StepSetThumbnail.Post", "Invalid path for non-user object (A)", step.Path))
				}
				return nil
			}

			// Standard path for all other records
			iconURL := builder.Permalink()
			iconURL = iconURL + "/attachments/" + attachment.AttachmentID.Hex()

			if err := schema.Set(object, step.Path, iconURL); err != nil {
				return Halt().WithError(derp.InternalError("build.StepSetThumbnail.Post", "Invalid path for non-user object (B)", step.Path))
			}
			return nil
		}
	}

	// Fall through means that we can't find any images.  Set the Thumbnail to an empty string.
	if err := schema.Set(object, step.Path, ""); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetThumbnail.Post", "Unable to set thumbnail"))
	}

	// Success!
	return nil
}
