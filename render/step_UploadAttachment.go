package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepUploadAttachment represents an action that can upload attachments.  It can only be used on a StreamRenderer
type StepUploadAttachment struct{}

func (step StepUploadAttachment) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepUploadAttachment) Post(renderer Renderer, _ io.Writer) error {

	// TODO: could this be generalized to work with more than just streams???
	factory := renderer.factory()
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

		if err := factory.MediaServer().Put(attachment.Filename, source); err != nil {
			return derp.Wrap(err, "whisper.handler.StepUploadAttachment.Post", "Error saving attachment to mediaserver", attachment)
		}

		if err := factory.Attachment().Save(&attachment, "Uploaded file: "+fileHeader.Filename); err != nil {
			return derp.Wrap(err, "whisper.handler.StepUploadAttachment.Post", "Error saving attachment", attachment)
		}
	}

	return nil
}
