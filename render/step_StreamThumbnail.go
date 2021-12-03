package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/davecgh/go-spew/spew"
)

// StepStreamThumbnail represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepStreamThumbnail struct {
	attachmentService *service.Attachment
}

func NewStepStreamThumbnail(attachmentService *service.Attachment, command datatype.Map) StepStreamThumbnail {

	return StepStreamThumbnail{
		attachmentService: attachmentService,
	}
}

// Get displays a form where users can update stream data
func (step StepStreamThumbnail) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepStreamThumbnail) Post(buffer io.Writer, renderer *Renderer) error {

	spew.Dump("Stream Thumbnail")
	// Find best icon from attachments
	attachments, err := step.attachmentService.ListByStream(renderer.stream.StreamID)

	if err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepStreamThumbnail.Post", "Error listing attachments")
	}

	attachment := new(model.Attachment)

	for attachments.Next(attachment) {
		spew.Dump(attachment)
		spew.Dump(attachment.MimeType())
		spew.Dump(attachment.MimeCategory())
		if attachment.MimeCategory() == "image" {
			renderer.stream.ThumbnailImage = attachment.Filename
			spew.Dump("Success!")
			return nil
		}
		attachment = new(model.Attachment)
	}

	spew.Dump("Done finding thumbnails :(")
	// Fall through to here means we should look at body content (but not now)

	return nil
}
