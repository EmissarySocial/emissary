package build

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/rosetta/translate"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepViewAttachment is a Step that can build a Stream into HTML
type StepViewAttachment struct {
	Categories sliceof.String     // Attachments must match one of these Categories to be accessible
	Formats    sliceof.String     // Allowed file types (e.g., "pdf", "docx")
	Widths     sliceof.Int        // The width(s) of the attachment (if image or video)
	Heights    sliceof.Int        // The height(s) of the attachment (if image or video)
	Bitrates   sliceof.Int        // The bitrate(s) of the attachment (if audio or video)
	Metadata   translate.Pipeline // Mapping to use when generating metadata
}

// Get builds the Stream HTML to the context
func (step StepViewAttachment) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepViewAttachment.Get"

	// Guarantee that this is a Stream
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.InternalError(location, "This step is only valid for Streams"))
	}

	// Check ETags to see if the browser already has a copy of this
	if matchHeader := streamBuilder.request().Header.Get("If-None-Match"); matchHeader == "1" {
		return Halt().WithStatusCode(http.StatusNotModified)
	}

	// Load the requested attachment from the database
	factory := streamBuilder.factory()
	objectID := streamBuilder.objectID()
	attachmentService := factory.Attachment()
	attachment := model.NewAttachment(model.AttachmentObjectTypeStream, objectID)

	switch attachmentIDString := streamBuilder.request().URL.Query().Get("attachmentId"); attachmentIDString {

	// Return the first matching attachment
	case "":

		var err error

		// Load the attachment record to verify that it is valid for this parent object
		attachment, err = attachmentService.LoadFirstByCategory(model.AttachmentObjectTypeStream, objectID, step.Categories)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error loading attachment", model.AttachmentObjectTypeStream, objectID, step.Categories))
		}

		// Search for a specific attachment
	default:

		attachmentID, err := primitive.ObjectIDFromHex(attachmentIDString)
		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid attachmentID", attachmentIDString))
		}

		// Load the attachment record to verify that it is valid for this parent object
		if err := attachmentService.LoadByID(model.AttachmentObjectTypeStream, objectID, attachmentID, &attachment); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error loading attachment", model.AttachmentObjectTypeStream, objectID, attachmentID))
		}
	}

	// RULE: Attachment must match the expected category
	if step.Categories.NotContains(attachment.Category) {
		return Halt().WithError(derp.NotFoundError(location, "Invalid attachment category: "+attachment.Category, derp.WithCode(http.StatusNotFound)))
	}

	// Retrieve the file from the mediaserver
	ms := factory.MediaServer()
	filespec, err := step.makeFileSpec(streamBuilder.request(), streamBuilder._stream, &attachment)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error generating file spec"))
	}

	if err := ms.Serve(streamBuilder.response(), streamBuilder.request(), filespec); err != nil {
		return Halt().WithError(derp.ReportAndReturn(derp.Wrap(err, location, "Error accessing attachment file")))
	}

	return Halt().AsFullPage()
}

func (step StepViewAttachment) Post(streamBuilder Builder, buffer io.Writer) PipelineBehavior {
	return Halt().WithError(derp.BadRequestError("build.StepViewAttachment.Post", "POST method not allowed for this step"))
}

// makeFileSpec generates a FileSpec for the given attachment based on the rules in this step and query parameters in the request
func (step StepViewAttachment) makeFileSpec(request *http.Request, stream *model.Stream, attachment *model.Attachment) (mediaserver.FileSpec, error) {

	const location = "build.StepViewAttachment.makeFileSpec"

	result := mediaserver.NewFileSpec()
	query := request.URL.Query()

	// Calculate generated file type
	if format := query.Get("format"); step.Formats.Contains(format) {
		result.Extension = "." + format
	} else {
		result.Extension = "." + step.Formats.First()
	}

	// (use canonical extension values.. sad face)
	switch result.Extension {
	case ".jpg":
		result.Extension = ".jpeg"
	}

	// Calculate valid image/video width
	if width := convert.Int(query.Get("width")); step.Widths.Contains(width) {
		result.Width = width
	} else {
		result.Width = step.Widths.First()
	}

	// Calculate valid image/video height
	if height := convert.Int(query.Get("height")); step.Heights.Contains(height) {
		result.Height = height
	} else {
		result.Height = step.Heights.First()
	}

	// Calculate valid audio/video bitrate
	if bitrate := convert.Int(query.Get("bitrate")); step.Bitrates.Contains(bitrate) {
		result.Bitrate = bitrate
	} else {
		result.Bitrate = step.Bitrates.First()
	}

	// Calculate metadata (if present)
	if len(step.Metadata) > 0 {

		inSchema := schema.New(model.StreamSchema())
		outSchema := schema.New(schema.Object{
			Wildcard: schema.String{},
		})

		if err := step.Metadata.Execute(inSchema, stream, outSchema, &result.Metadata); err != nil {
			return result, derp.Wrap(err, location, "Error executing metadata pipeline")
		}
	}

	// Set other properties
	result.Filename = attachment.AttachmentID.Hex()
	result.OriginalExtension = attachment.OriginalExtension()
	result.Cache = true

	return result, nil
}
