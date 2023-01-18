package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/path"
)

// StepSetThumbnail represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepSetThumbnail struct {
	Path string
}

func (step StepSetThumbnail) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepSetThumbnail) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepSetThumbnail) Post(renderer Renderer) error {

	// Find best icon from attachments
	factory := renderer.factory()

	objectType := renderer.service().ObjectType()
	objectID := renderer.objectID()

	attachments, err := factory.Attachment().QueryByObjectID(objectType, objectID)

	if err != nil {
		return derp.NewBadRequestError("render.StepSetThumbnail.Post", "Error listing attachments")
	}

	// Scan all attachments and use the first one that is an image.
	for _, attachment := range attachments {
		if attachment.MimeCategory() == "image" {

			// Special case for User objects (this should always be "imageId")
			if objectType == "User" {
				if ok := path.SetString(renderer.object(), step.Path, attachment.AttachmentID.Hex()); ok {
					return nil
				} else {
					return derp.NewInternalError("render.StepSetThumbnail.Post", "Invalid path for non-user object (A)", step.Path)
				}
			}

			// Standard path for all other records
			imageURL := renderer.Permalink()
			imageURL = imageURL + "/attachments/" + attachment.AttachmentID.Hex()

			if ok := path.SetString(renderer.object(), step.Path, imageURL); ok {
				return nil
			} else {
				return derp.NewInternalError("render.StepSetThumbnail.Post", "Invalid path for non-user object (B)", step.Path)
			}
		}
	}

	path.SetString(renderer.object(), step.Path, "")
	return nil
}
