package step

import (
	"github.com/benpate/rosetta/mapof"
)

// UploadAttachment represents an action that can upload attachments.  It can only be used on a StreamBuilder
type UploadAttachment struct {
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

// NewUploadAttachment returns a fully parsed UploadAttachment object
func NewUploadAttachment(stepInfo mapof.Any) (UploadAttachment, error) {

	// Default behavior is "append".  Only other option is "replace"
	action := stepInfo.GetString("action")

	if action != "replace" {
		action = "append"
	}

	return UploadAttachment{
		Action:         action,
		Fieldname:      first(stepInfo.GetString("fieldname"), "form"),
		AttachmentPath: stepInfo.GetString("attachment-path"),
		DownloadPath:   stepInfo.GetString("download-path"),
		FilenamePath:   stepInfo.GetString("filename-path"),
		AcceptType:     stepInfo.GetString("accept-type"),
		Maximum:        max(stepInfo.GetInt("maximum"), 0),
		Category:       stepInfo.GetString("category"),
		JSONResult:     stepInfo.GetBool("json-result"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step UploadAttachment) AmStep() {}
