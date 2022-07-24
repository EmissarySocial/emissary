package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/path"
	"github.com/davecgh/go-spew/spew"
)

// StepSetThumbnail represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepSetThumbnail struct{}

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

	spew.Dump("Finding Thumbnail for", renderer.object())
	attachments, err := factory.Attachment().ListByObjectID(renderer.objectID())

	if err != nil {
		return derp.NewBadRequestError("render.StepSetThumbnail.Post", "Error listing attachments")
	}

	// Scan all attachments and use the first one that is an image.
	attachment := model.NewAttachment(renderer.objectID())
	for attachments.Next(&attachment) {

		spew.Dump("checking", attachment)
		if attachment.MimeCategory() == "image" {
			spew.Dump("Success")
			err := path.Set(renderer.object(), "thumbnailImage", attachment.AttachmentID.Hex())
			spew.Dump(err, renderer.object())
			return err
		}
		attachment = model.NewAttachment(renderer.objectID())
	}

	// Fall through to here means we should look at body content (but not now)
	// So, for now, if there's no thumbnail, then set "" as default
	return path.Set(renderer.object(), "thumbnailImage", "")

}
