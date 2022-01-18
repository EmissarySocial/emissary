package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/whisperverse/mediaserver"
	"github.com/whisperverse/whisperverse/service"
)

// StepUploadAttachment represents an action that can upload attachments.  It can only be used on a StreamRenderer
type StepUploadAttachment struct {
	streamService     *service.Stream
	attachmentService *service.Attachment
	mediaServer       mediaserver.MediaServer
}

// NewStepUploadAttachment returns a fully parsed StepUploadAttachment object
func NewStepUploadAttachment(streamService *service.Stream, attachmentService *service.Attachment, mediaServer mediaserver.MediaServer, config datatype.Map) StepUploadAttachment {

	return StepUploadAttachment{
		streamService:     streamService,
		attachmentService: attachmentService,
		mediaServer:       mediaServer,
	}
}

func (step StepUploadAttachment) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

func (step StepUploadAttachment) Post(buffer io.Writer, renderer Renderer) error {

	// TODO: could this be generalized to work with more than just streams???
	streamRenderer := renderer.(*Stream)
	form, err := streamRenderer.ctx.MultipartForm()

	if err != nil {
		return derp.Wrap(err, "whisper.handler.StepUploadAttachment.Post", "Error reading multipart form.")
	}

	files := form.File["file"]

	for _, fileHeader := range files {

		// Each attachment is tracked separately, so make a new attachment for each file in the upload.
		attachment := streamRenderer.stream.NewAttachment(fileHeader.Filename)

		// Open the source (from the POST request)
		source, err := fileHeader.Open()

		if err != nil {
			return derp.Wrap(err, "whisper.handler.StepUploadAttachment.Post", "Error reading file from multi-part header", fileHeader)
		}

		defer source.Close()

		if err := step.mediaServer.Put(attachment.Filename, source); err != nil {
			return derp.Wrap(err, "whisper.handler.StepUploadAttachment.Post", "Error saving attachment to mediaserver", attachment)
		}

		if err := step.attachmentService.Save(&attachment, "Uploaded file: "+fileHeader.Filename); err != nil {
			return derp.Wrap(err, "whisper.handler.StepUploadAttachment.Post", "Error saving attachment", attachment)
		}
	}

	return nil
}
