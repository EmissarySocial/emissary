package build

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"strconv"
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
	Action         string // Action to perform when uploading the attachment ("append", "replace", or "fill")
	Fieldname      string // Name of the form field that contains the file data (Default: "file")
	AttachmentPath string // Path name to store the AttachmentID
	DownloadPath   string // Path name to store the download URL
	FilenamePath   string // Path name to store the original filename

	AcceptType string // Mime Type(s) to accept (e.g. "image/*")
	Category   string // Category to apply to the Attachment
	Maximum    int    // Maximum number of uploads to allow (Default: 1)
	Minimum    int    // Fewest attachments the category may hold once the upload is done (Default: 0)
	JSONResult bool   // If TRUE, return a JSON structure with result data. This forces Maximum=1

	Label                string // Value to set as the attachment.label
	LabelFieldname       string // Form field that defines the attachment label
	Description          string // Value to set as the attachment.description
	DescriptionFieldname string // Form field that defines the attachment description

	RuleHeight int      // Fixed height for all downloads
	RuleWidth  int      // Fixed width for all downloads
	RuleTypes  []string // Allowed extensions.  The first value is used as the default.
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepUploadAttachments) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepUploadAttachments) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepUploadAttachments.Post"

	// Read the multipart form from the request
	form, err := multipartForm(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Reading multipart form."))
	}

	// Retrieve upload files from the POST
	files := form.File[step.Fieldname]

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

	// Load what the category already holds: the count decides what fits, and the ranks decide where new files go
	existing, err := attachmentService.QueryByCategory(builder.session(), objectType, objectID, step.Category)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Loading existing Attachments"))
	}

	// Number of files must be less or equal to the maximum.  "fill" keeps every existing
	// attachment, so it accepts only as many new files as there is room left for.
	limit := step.Maximum

	if step.Action == "fill" {
		limit = roomLeft(step.Maximum, len(existing))
	}

	if len(files) > limit {
		files = files[:limit]
	}

	// RULE: The category must hold at least the minimum once these files are stored
	if count := countAfterUpload(step.Action, step.Maximum, len(existing), len(files)); count < step.Minimum {
		return Halt().WithError(derp.Validation(minimumMessage(step.Minimum), "minimum", step.Minimum, "count", count))
	}

	if len(files) == 0 {
		return Continue()
	}

	// Sniff every file before storing any, so a file the step refuses halts with nothing written
	uploads, err := openUploads(files, step.AcceptType)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Checking uploaded files"))
	}

	defer closeUploads(uploads)

	// Make room for new attachments.  For "fill", the files already fit, so nothing is deleted.
	if err := attachmentService.MakeRoom(builder.session(), objectType, objectID, step.Category, step.Action, step.Maximum, len(files)); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Making room for new Attachments"))
	}

	// New files rank after everything the category holds, in the order posted
	nextRank := nextRankAfter(step.Action, existing)

	// Make attachments for each uploaded file
	for index, upload := range uploads {

		log.Trace().Str("Filename", upload.header.Filename).Msg("Found file")

		// Create a new Attachment object
		attachment := model.NewAttachment(objectType, objectID)
		attachment.Original = upload.header.Filename
		attachment.ContentType = upload.contentType
		attachment.Category = step.Category
		attachment.Rank = nextRank + index

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
		if err := factory.MediaServer().Put(attachment.AttachmentID.Hex(), upload.reader); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Saving attachment to mediaserver", attachment))
		}

		// Apply rules to Attachment
		attachment.SetRules(step.RuleWidth, step.RuleHeight, step.RuleTypes)

		// Try to save the Attachment
		if err := attachmentService.Save(builder.session(), &attachment, "Uploaded file: "+upload.header.Filename); err != nil {
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

// roomLeft returns how many more attachments fit beneath the maximum, when there are already
// "existing" of them.  It is never negative, even when a lower maximum leaves too many already.
func roomLeft(maximum int, existing int) int {
	return max(maximum-existing, 0)
}

// countAfterUpload returns how many attachments the category will hold once "posted" new files
// are stored under the action: "replace" keeps only the new files, "fill" adds them to the
// existing ones (the caller has already trimmed them to the room left), and "append" adds them
// and then deletes whatever overflows the maximum.
func countAfterUpload(action string, maximum int, existing int, posted int) int {

	switch action {

	case "replace":
		return posted

	case "fill":
		return existing + posted

	default:
		return min(existing+posted, maximum)
	}
}

// minimumMessage returns the message shown inline when an upload would leave the category
// holding fewer files than the step requires
func minimumMessage(minimum int) string {

	if minimum == 1 {
		return "At least 1 file is required"
	}

	return "At least " + strconv.Itoa(minimum) + " files are required"
}

// upload is one posted file, opened and sniffed, ready to store
type upload struct {
	header      *multipart.FileHeader
	source      multipart.File
	reader      io.Reader
	contentType string
}

// openUploads opens and sniffs every posted file before any is stored, so a file the step refuses
// halts the upload with nothing written to the media server. The caller closes what it returns.
func openUploads(files []*multipart.FileHeader, acceptType string) ([]upload, error) {

	const location = "build.openUploads"

	result := make([]upload, 0, len(files))

	for _, header := range files {

		// Open the uploaded file contents
		source, err := header.Open()

		if err != nil {
			closeUploads(result)
			return nil, derp.Wrap(err, location, "Reading file from multi-part header", header.Filename)
		}

		// Sniff the actual file contents (NOT the attacker-controlled filename or
		// Content-Type header) and, if this step restricts the accepted content-types,
		// reject anything that does not match.  `reader` re-assembles the bytes we
		// peeked so the full stream is still available for MediaServer.Put.
		reader, contentType, err := verifyContentType(source, header.Filename, acceptType)

		if err != nil {
			derp.Report(source.Close())
			closeUploads(result)
			return nil, derp.Wrap(err, location, "Uploaded file is not an allowed type", header.Filename, acceptType)
		}

		result = append(result, upload{header: header, source: source, reader: reader, contentType: contentType})
	}

	return result, nil
}

// closeUploads closes the files that openUploads opened
func closeUploads(uploads []upload) {
	for _, upload := range uploads {
		derp.Report(upload.source.Close())
	}
}

// nextRankAfter returns the rank the first new file takes: 1 when the action replaces
// everything, and otherwise one past the highest rank the category holds
func nextRankAfter(action string, existing []model.Attachment) int {

	if action == "replace" {
		return 1
	}

	if len(existing) == 0 {
		return 1
	}

	// The category is in rank order, so its last file holds the highest rank; a rank below 1 is unset
	return max(existing[len(existing)-1].Rank, 0) + 1
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
