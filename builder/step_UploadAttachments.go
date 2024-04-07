package builder

import (
	"encoding/json"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog/log"
)

// StepUploadAttachments represents an action that can upload attachments.  It can only be used on a StreamBuilder
type StepUploadAttachments struct {
	Action         string // Action to perform when uploading the attachment ("append" or "replace")
	Fieldname      string // Name of the form field that contains the file data (Default: "file")
	AttachmentPath string // Path name to store the AttachmentID
	DownloadPath   string // Path name to store the download URL
	FilenamePath   string // Path name to store the original filename
	AcceptType     string // Mime Type(s) to accept (e.g. "image/*")
	Category       string // Category to apply to the Attachment
	Maximum        int    // Maximum number of uploads to allow (Default: 1)
	JSONResult     bool   // If TRUE, return a JSON structure with result data. This forces Maximum=1
}

func (step StepUploadAttachments) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

func (step StepUploadAttachments) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "handler.StepUploadAttachments.Post"

	log.Debug().Msg(location)

	// Read the multipart form from the request
	form, err := multipartForm(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error reading multipart form."))
	}

	files := form.File[step.Fieldname]

	log.Trace().Str("fieldname", step.Fieldname).Msg("Searching Form for Files")

	if len(files) == 0 {
		return Continue()
	}

	/*/ Auto-detect EditorJS
	if len(files) == 0 {
		files = form.File["image"]
		isEditorJS = true
		step.Maximum = 1
	}*/

	factory := builder.factory()
	attachmentService := factory.Attachment()

	object := builder.object()
	objectID := builder.objectID()
	objectType := builder.service().ObjectType()

	// Special case:  If we're uploading a draft, then we need to attach the document to the parent stream.
	if objectType == "StreamDraft" {
		objectType = "Stream"
	}

	// Number of files must be less or equal to the maximum
	if len(files) > step.Maximum {
		files = files[:step.Maximum]
	}

	// Make room for new attachments
	if err := attachmentService.MakeRoom(objectType, objectID, step.Category, step.Action, step.Maximum, len(files)); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error making room for new Attachments"))
	}

	// Make attachments for each uploaded file
	for _, fileHeader := range files {

		log.Trace().Str("Filename", fileHeader.Filename).Msg("Found file")

		attachment := model.NewAttachment(objectType, objectID)
		attachment.Original = fileHeader.Filename
		attachment.Category = step.Category

		// Open the source (from the POST request)
		source, err := fileHeader.Open()

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error reading file from multi-part header", fileHeader))
		}

		defer source.Close()

		// Add the image into the media server
		width, height, err := factory.MediaServer().Put(attachment.AttachmentID.Hex(), source)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error saving attachment to mediaserver", attachment))
		}

		// Update original dimensions
		attachment.Width = width
		attachment.Height = height

		// Try to save the Attachment
		if err := attachmentService.Save(&attachment, "Uploaded file: "+fileHeader.Filename); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error saving attachment", attachment))
		}

		// Try to put the the attachmentId into the object
		if step.AttachmentPath != "" {
			log.Trace().Str("AttachmentPath", step.AttachmentPath).Str("Value", attachment.AttachmentID.Hex()).Msg("Setting attachment path")
			if err := builder.schema().Set(object, step.AttachmentPath, attachment.AttachmentID.Hex()); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Error setting download path", attachment))
			}
		}

		// Try to put the the downloadUrl into the object
		if step.DownloadPath != "" {
			log.Trace().Str("DownloadPath", step.DownloadPath).Str("Value", attachment.URL).Msg("Setting download path")
			if err := builder.schema().Set(object, step.DownloadPath, attachment.URL); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Error setting download path", attachment))
			}
		}

		// Try to put the original filename into the object
		if step.FilenamePath != "" {
			log.Trace().Str("FilenamePath", step.FilenamePath).Str("Value", attachment.Original).Msg("Setting filename path")
			if err := builder.schema().Set(object, step.FilenamePath, attachment.Original); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Error setting filename path", attachment))
			}
		}

		// EditorJS can only upload a single file at a time.
		if step.JSONResult {
			response := mapof.Any{
				"success": 1,
				"file": mapof.Any{
					"url":    attachment.CalcURL(builder.Host()),
					"height": attachment.Height,
					"width":  attachment.Width,
				},
				"data": mapof.Any{
					"filePath": attachment.CalcURL(builder.Host()),
				},
			}

			// Marshal the response into JSON
			bytes, err := json.Marshal(response)

			if err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Error marshalling response", response))
			}

			// Write the response to the buffer
			if _, err := buffer.Write(bytes); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Error writing response to buffer", response))
			}

			// Tell the client that we're done.
			return Continue().AsFullPage().WithContentType("application/json")
		}
	}

	// After all files are uploaded, tell the client that we're done.
	return Continue().WithEvent("attachments-updated", "true")
}
