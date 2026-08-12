package build

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/rs/zerolog/log"
)

// StepUploadAttachments represents an action that can upload attachments.  It can only be used on a StreamBuilder
type StepUploadAttachments struct {
	Action         string // Action to perform when uploading the attachment ("append" or "replace")
	Fieldname      string // Name of the form field that contains the file data (Default: "file")
	AttachmentPath string // Path name to store the AttachmentID
	DownloadPath   string // Path name to store the download URL
	FilenamePath   string // Path name to store the original filename

	AcceptType string // Mime Type(s) to accept (e.g. "image/*")
	Category   string // Category to apply to the Attachment
	Maximum    int    // Maximum number of uploads to allow (Default: 1)
	JSONResult bool   // If TRUE, return a JSON structure with result data. This forces Maximum=1

	Label                string // Value to set as the attachment.label
	LabelFieldname       string // Form field that defines the attachment label
	Description          string // Value to set as the attachment.description
	DescriptionFieldname string // Form field that defines the attachment description

	RuleHeight int      // Fixed height for all downloads
	RuleWidth  int      // Fixed width for all downloads
	RuleTypes  []string // Allowed extensions.  The first value is used as the default.
}

func (step StepUploadAttachments) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

func (step StepUploadAttachments) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "handler.StepUploadAttachments.Post"

	// Read the multipart form from the request
	form, err := multipartForm(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Reading multipart form."))
	}

	// Retrieve upload files from the POST
	files := form.File[step.Fieldname]

	if len(files) == 0 {
		return Continue()
	}

	// Number of files must be less or equal to the maximum
	if len(files) > step.Maximum {
		files = files[:step.Maximum]
	}

	// Required services and objects
	factory := builder.factory()
	attachmentService := factory.Attachment()

	object := builder.object()
	objectID := builder.objectID()
	objectType := builder.service().ObjectType()

	// Special case:  If we're uploading a draft, then we need to attach the document to the parent stream.
	if objectType == "StreamDraft" {
		objectType = "Stream"
	}

	// Make room for new attachments
	if err := attachmentService.MakeRoom(builder.session(), objectType, objectID, step.Category, step.Action, step.Maximum, len(files)); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Making room for new Attachments"))
	}

	// Make attachments for each uploaded file
	for index, fileHeader := range files {

		log.Trace().Str("Filename", fileHeader.Filename).Msg("Found file")

		// Open the uploaded file contents
		source, err := fileHeader.Open()

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Reading file from multi-part header", fileHeader))
		}

		defer derp.ReportFunc(source.Close)

		// Sniff the actual file contents (NOT the attacker-controlled filename or
		// Content-Type header) and, if this step restricts the accepted content-types,
		// reject anything that does not match.  `reader` re-assembles the bytes we
		// peeked so the full stream is still available for MediaServer.Put.
		reader, contentType, err := verifyContentType(source, fileHeader.Filename, step.AcceptType)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Uploaded file is not an allowed type", fileHeader.Filename, step.AcceptType))
		}

		// Create a new Attachment object
		attachment := model.NewAttachment(objectType, objectID)
		attachment.Original = fileHeader.Filename
		attachment.ContentType = contentType
		attachment.Category = step.Category

		// Try to set labels from the stepInfo and form
		if step.Label != "" {
			attachment.Label = step.Label
		} else if step.LabelFieldname != "" {
			attachment.Label = slice.At(form.Value[step.LabelFieldname], index)
		}

		// Try to set descriptions from the stepInfo and form
		if step.Description != "" {
			attachment.Description = step.Description
		} else if step.DescriptionFieldname != "" {
			attachment.Description = slice.At(form.Value[step.DescriptionFieldname], index)
		}

		// Add the document into the media server.
		// If it's an image or video, then save the dimensions as well.
		if err := factory.MediaServer().Put(attachment.AttachmentID.Hex(), reader); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Saving attachment to mediaserver", attachment))
		}

		// Apply rules to Attachment
		attachment.SetRules(step.RuleWidth, step.RuleHeight, step.RuleTypes)

		// Try to save the Attachment
		if err := attachmentService.Save(builder.session(), &attachment, "Uploaded file: "+fileHeader.Filename); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Saving attachment", attachment))
		}

		// Try to put the the attachmentId into the object
		if step.AttachmentPath != "" {
			if err := builder.schema().Set(object, step.AttachmentPath, attachment.AttachmentID.Hex()); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Setting attachment path", attachment))
			}
		}

		// Try to put the the downloadUrl into the object
		if step.DownloadPath != "" {
			if err := builder.schema().Set(object, step.DownloadPath, attachment.URL); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Setting download path", attachment))
			}
		}

		// Try to put the original filename into the object
		if step.FilenamePath != "" {
			if err := builder.schema().Set(object, step.FilenamePath, attachment.Original); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Setting filename path", attachment))
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
				return Halt().WithError(derp.Wrap(err, location, "Marshalling response", response))
			}

			// Write the response to the buffer
			if _, err := buffer.Write(bytes); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Writing response to buffer", response))
			}

			// Tell the client that we're done.
			return Continue().AsFullPage().WithContentType("application/json")
		}
	}

	// After all files are uploaded, tell the client that we're done.
	return Continue().WithEvent("attachments-updated", "true")
}

// verifyContentType sniffs an uploaded file to determine its actual content-type, and
// confirms that it matches the provided accept pattern (for instance "image/*" or
// "image/png,image/webp").
func verifyContentType(source io.Reader, filename string, acceptType string) (io.Reader, string, error) {

	const location = "build.verifyContentType"

	// The returned reader replays the sniffed bytes followed by the rest of the file, so
	// callers can still consume the entire stream.  The detected type is returned even when
	// acceptType is empty ("allow anything"), because it is recorded on the Attachment and
	// decides how the file may be served later.  The filename never decides the type; it
	// only picks audio-vs-video inside containers the bytes have already confirmed.

	// Peek at the first 512 bytes -- the amount http.DetectContentType inspects.
	header := make([]byte, 512)
	headerLength, err := io.ReadFull(source, header)

	if err != nil {

		// io.EOF / io.ErrUnexpectedEOF simply mean the file is shorter than 512 bytes, which is
		// fine.  Any other error is a genuine read failure.
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, "", derp.Wrap(err, location, "Reading uploaded file")
		}
	}

	header = header[:headerLength]

	// Reassemble the full stream: the bytes we peeked, followed by whatever remains.
	reader := io.MultiReader(bytes.NewReader(header), source)

	// Sniff the actual content-type from the file's own bytes.
	detected := model.DetectContentType(header, filename)

	// An empty accept pattern means "allow anything" -- preserve legacy behavior.
	if acceptType == "" {
		return reader, detected, nil
	}

	// RULE: The file's contents must match the types this step accepts.
	if !contentTypeMatches(detected, acceptType) {
		return nil, "", derp.BadRequest(location, "Uploaded file type is not allowed", detected, acceptType)
	}

	return reader, detected, nil
}

// contentTypeMatches reports whether a detected content-type (e.g. "image/png")
// satisfies an accept pattern.  The pattern may be a comma-separated list of media
// ranges, each either an exact type ("image/png") or a wildcard ("image/*", "*/*").
func contentTypeMatches(detected string, acceptType string) bool {

	// DetectContentType may return "type/subtype; charset=..." -- keep only the type.
	detected = strings.TrimSpace(list.Semicolon(detected).First())

	for _, pattern := range strings.Split(acceptType, ",") {

		pattern = strings.TrimSpace(pattern)

		switch {

		case pattern == "" || pattern == "*/*":
			return true

		// "image/*" matches any subtype within the "image" category.
		case strings.HasSuffix(pattern, "/*"):
			if list.Slash(detected).First() == list.Slash(pattern).First() {
				return true
			}

		// Otherwise require an exact media-type match.
		case detected == pattern:
			return true
		}
	}

	return false
}
