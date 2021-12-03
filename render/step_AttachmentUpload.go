package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/davecgh/go-spew/spew"
)

// StepAttachmentUpload represents an action that can edit a top-level folder in the Domain
type StepAttachmentUpload struct {
	streamService     *service.Stream
	attachmentService *service.Attachment
}

// NewStepAttachmentUpload returns a fully parsed StepAttachmentUpload object
func NewStepAttachmentUpload(streamService *service.Stream, attachmentService *service.Attachment, config datatype.Map) StepAttachmentUpload {

	return StepAttachmentUpload{
		streamService:     streamService,
		attachmentService: attachmentService,
	}
}

func (step StepAttachmentUpload) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step StepAttachmentUpload) Post(buffer io.Writer, renderer *Renderer) error {

	spew.Dump("Attachment Upload")

	form, err := renderer.ctx.MultipartForm()

	if err != nil {
		return derp.Wrap(err, "ghost.handler.StepAttachmentUpload.Post", "Error reading multipart form.")
	}

	filesystem := step.attachmentService.Filesystem()
	files := form.File["file"]

	for _, fileHeader := range files {

		// Each attachment is tracked separately, so make a new attachment for each file in the upload.
		attachment := renderer.stream.NewAttachment()
		attachment.Original = fileHeader.Filename
		attachment.Filename = attachment.AttachmentID.Hex()

		// Open the source (from the POST request)
		source, err := fileHeader.Open()

		if err != nil {
			return derp.Wrap(err, "ghost.handler.StepAttachmentUpload.Post", "Error reading file from multi-part header", fileHeader)
		}

		defer source.Close()

		// Open the destination (in afero)
		destination, err := filesystem.Create(attachment.Filename)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.StepAttachmentUpload.Post", "Error creating file in filesystem", attachment)
		}

		defer destination.Close()

		// Save the upload into the destination
		if _, err = io.Copy(destination, source); err != nil {
			return derp.Wrap(err, "ghost.handler.StepAttachmentUpload.Post", "Error writing attachment file", attachment, fileHeader)
		}

		if err := step.attachmentService.Save(&attachment, "Uploaded file: "+fileHeader.Filename); err != nil {
			return derp.Wrap(err, "ghost.handler.StepAttachmentUpload.Post", "Error saving attachment", attachment)
		}
	}

	return nil
}
