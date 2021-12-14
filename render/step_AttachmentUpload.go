package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/benpate/mediaserver"
)

// StepAttachmentUpload represents an action that can upload attachments.  It can only be used on a StreamRenderer
type StepAttachmentUpload struct {
	streamService     *service.Stream
	attachmentService *service.Attachment
	mediaServer       mediaserver.MediaServer
}

// NewStepAttachmentUpload returns a fully parsed StepAttachmentUpload object
func NewStepAttachmentUpload(streamService *service.Stream, attachmentService *service.Attachment, mediaServer mediaserver.MediaServer, config datatype.Map) StepAttachmentUpload {

	return StepAttachmentUpload{
		streamService:     streamService,
		attachmentService: attachmentService,
		mediaServer:       mediaServer,
	}
}

func (step StepAttachmentUpload) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

func (step StepAttachmentUpload) Post(buffer io.Writer, renderer Renderer) error {

	streamRenderer := renderer.(Stream)
	form, err := streamRenderer.ctx.MultipartForm()

	if err != nil {
		return derp.Wrap(err, "ghost.handler.StepAttachmentUpload.Post", "Error reading multipart form.")
	}

	files := form.File["file"]

	for _, fileHeader := range files {

		// Each attachment is tracked separately, so make a new attachment for each file in the upload.
		attachment := streamRenderer.stream.NewAttachment(fileHeader.Filename)

		// Open the source (from the POST request)
		source, err := fileHeader.Open()

		if err != nil {
			return derp.Wrap(err, "ghost.handler.StepAttachmentUpload.Post", "Error reading file from multi-part header", fileHeader)
		}

		defer source.Close()

		if err := step.mediaServer.Put(attachment.Filename, source); err != nil {
			return derp.Wrap(err, "ghost.handler.StepAttachmentUpload.Post", "Error saving attachment to mediaserver", attachment)
		}

		if err := step.attachmentService.Save(&attachment, "Uploaded file: "+fileHeader.Filename); err != nil {
			return derp.Wrap(err, "ghost.handler.StepAttachmentUpload.Post", "Error saving attachment", attachment)
		}
	}

	return nil
}
