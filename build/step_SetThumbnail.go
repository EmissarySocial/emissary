package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepSetThumbnail is a Step that writes the object's first image attachment into a schema path
type StepSetThumbnail struct {
	Path     string
	Category string
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepSetThumbnail) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post writes the first image among the object's attachments into the step's path.
func (step StepSetThumbnail) Post(builder Builder, _ io.Writer) PipelineBehavior {

	// Find best icon from attachments, within the step's category when it names one
	factory := builder.factory()

	objectType := builder.service().ObjectType()
	objectID := builder.objectID()
	object := builder.object()

	attachments, err := factory.Attachment().QueryByCategory(builder.session(), objectType, objectID, step.Category)

	if err != nil {
		return Halt().WithError(derp.BadRequest("build.StepSetThumbnail.Post", "Unable to list attachments"))
	}

	// Scan the attachments in rank order and use the first one that is an image.

	schema := builder.schema()

	for _, attachment := range attachments {
		if attachment.MimeCategory() == "image" {

			// Special case for User objects (this should always be "iconId")
			if objectType == "User" {
				if err := schema.Set(object, step.Path, attachment.AttachmentID.Hex()); err != nil {
					return Halt().WithError(derp.Internal("build.StepSetThumbnail.Post", "Invalid path for non-user object (A)", step.Path))
				}
				return nil
			}

			// Standard path for all other records
			iconURL := builder.Permalink()
			iconURL = iconURL + "/attachments/" + attachment.AttachmentID.Hex()

			if err := schema.Set(object, step.Path, iconURL); err != nil {
				return Halt().WithError(derp.Internal("build.StepSetThumbnail.Post", "Invalid path for non-user object (B)", step.Path))
			}
			return nil
		}
	}

	// Fall through means that we can't find any images.  Set the Thumbnail to an empty string.
	if err := schema.Set(object, step.Path, ""); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetThumbnail.Post", "Setting thumbnail"))
	}

	// Success!
	return nil
}
