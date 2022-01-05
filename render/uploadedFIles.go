package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// uploadedFiles extracts
func uploadedFiles(factory Factory, ctx echo.Context, objectID primitive.ObjectID) []model.Attachment {

	result := make([]model.Attachment, 0)
	form, err := ctx.MultipartForm()

	if err != nil {
		return result // "Silent" failure just skips forms that cannot be read
	}

	// Get all files from the multi-part form
	files := form.File["file"]

	if len(files) == 0 {
		return result // skip empty forms, too
	}

	// Now that we know we're (probably) going to upload some files,
	// it's time to break out the media server.
	mediaServer := factory.MediaServer()
	attachmentService := factory.Attachment()

	// Create new attachments for each uploaded file
	for _, fileHeader := range files {

		// Each attachment is tracked separately, so make a new attachment for each file in the upload.
		attachment := model.NewAttachment(objectID)
		attachment.Original = fileHeader.Filename

		// Open the source (from the POST request)
		source, err := fileHeader.Open()

		if err != nil {
			derp.Report(err)
			continue // if we can't open/read the uploaded file, then just skip it.
		}

		defer source.Close()

		if err := mediaServer.Put(attachment.Filename, source); err != nil {
			derp.Report(derp.Wrap(err, "ghost.handler.StepUploadAttachment.Post", "Error saving attachment to mediaserver", attachment))
			continue // semi-silent failure
		}

		if err := attachmentService.Save(&attachment, "Uploaded file: "+fileHeader.Filename); err != nil {
			derp.Report(derp.Wrap(err, "ghost.handler.StepUploadAttachment.Post", "Error saving attachment", attachment))
			continue // semi-silent failure
		}

		result = append(result, attachment)
	}

	return result
}
