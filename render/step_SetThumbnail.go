package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/service"
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
func (step StepStreamThumbnail) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepStreamThumbnail) Post(buffer io.Writer, renderer Renderer) error {

	// Find best icon from attachments
	attachments, err := step.attachmentService.ListByObjectID(renderer.objectID())

	if err != nil {
		return derp.New(derp.CodeBadRequestError, "whisper.render.StepStreamThumbnail.Post", "Error listing attachments")
	}

	// Scan all attachments and use the first one that is an image.
	attachment := new(model.Attachment)
	for attachments.Next(attachment) {

		if attachment.MimeCategory() == "image" {
			return path.Set(renderer, "thumbnailImage", attachment.Filename)
		}
		attachment = new(model.Attachment)
	}

	// Fall through to here means we should look at body content (but not now)

	return nil
}
