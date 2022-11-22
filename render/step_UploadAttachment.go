package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/maps"
)

// StepUploadAttachment represents an action that can upload attachments.  It can only be used on a StreamRenderer
type StepUploadAttachment struct {
	Maximum int
}

func (step StepUploadAttachment) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepUploadAttachment) UseGlobalWrapper() bool {
	return true
}

func (step StepUploadAttachment) Post(renderer Renderer) error {

	factory := renderer.factory()
	context := renderer.context()

	form, err := context.MultipartForm()

	if err != nil {
		return derp.Wrap(err, "handler.StepUploadAttachment.Post", "Error reading multipart form.")
	}

	files := form.File["file"]
	isEditorJS := false

	// Auto-detect EditorJS
	if len(files) == 0 {
		files = form.File["image"]
		isEditorJS = true
		step.Maximum = 1
	}

	attachmentService := factory.Attachment()

	objectType := renderer.service().ObjectType()
	objectID := renderer.objectID()

	for index, fileHeader := range files {

		attachment := model.NewAttachment(objectType, objectID)
		attachment.Original = fileHeader.Filename

		// Open the source (from the POST request)
		source, err := fileHeader.Open()

		if err != nil {
			return derp.Wrap(err, "handler.StepUploadAttachment.Post", "Error reading file from multi-part header", fileHeader)
		}

		defer source.Close()

		// Add the image into the media server
		width, height, err := factory.MediaServer().Put(attachment.AttachmentID.Hex(), source)

		if err != nil {
			return derp.Wrap(err, "handler.StepUploadAttachment.Post", "Error saving attachment to mediaserver", attachment)
		}

		// Update original dimensions
		attachment.Width = width
		attachment.Height = height

		// Try to save
		if err := attachmentService.Save(&attachment, "Uploaded file: "+fileHeader.Filename); err != nil {
			return derp.Wrap(err, "handler.StepUploadAttachment.Post", "Error saving attachment", attachment)
		}

		// EditorJS can only upload a single file at a time.
		if isEditorJS {
			response := maps.Map{
				"success": 1,
				"file": maps.Map{
					"url":    attachment.URL(),
					"height": attachment.Height,
					"width":  attachment.Width,
				},
			}

			return renderer.context().JSON(200, response)
		}

		// If we have reached the maximum configured number of attachments,
		// then stop processing here.
		if (step.Maximum > 0) && index >= step.Maximum {
			break
		}
	}

	// After all files are uploaded, tell the client that we're done.
	renderer.context().Response().Header().Set("HX-Trigger", `attachments-updated`)

	return nil
}
