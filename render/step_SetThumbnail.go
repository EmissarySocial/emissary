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

			imageURL := renderer.Permalink()

			if objectType == "User" {
				imageURL = imageURL + "/avatar/" + attachment.AttachmentID.Hex()
			} else {
				imageURL = imageURL + "/attachments/" + attachment.AttachmentID.Hex()
			}

			err := path.Set(renderer.object(), step.Path, imageURL)
			return err
		}
	}

	// Fall through to here means we should look at body content (but not now)
	// So, for now, if there's no thumbnail, then set "" as default
	return path.Set(renderer.object(), step.Path, "")
}
