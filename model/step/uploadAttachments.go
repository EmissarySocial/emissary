package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// UploadAttachments represents an action that can upload attachments.  It can only be used on a StreamBuilder
type UploadAttachments struct {
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

// NewUploadAttachments returns a fully parsed UploadAttachments object
func NewUploadAttachments(stepInfo mapof.Any) (UploadAttachments, error) {

	const location = "step.NewUploadAttachments"

	// Default behavior is "append", which deletes the oldest attachments to make room for new ones.
	// "replace" deletes them all, and "fill" deletes none, adding only as many new ones as fit.
	action := stepInfo.GetString("action")

	if action != "replace" && action != "fill" {
		action = "append"
	}

	maximum := max(stepInfo.GetInt("maximum"), 1)
	minimum := max(stepInfo.GetInt("minimum"), 0)

	// RULE: A minimum above the maximum could never be met
	if minimum > maximum {
		return UploadAttachments{}, derp.BadRequest(location, "The 'minimum' attribute cannot exceed the 'maximum'", minimum, maximum)
	}

	rules := stepInfo.GetMap("rules")

	return UploadAttachments{
		Action:         action,
		Fieldname:      first(stepInfo.GetString("fieldname"), "file"),
		AttachmentPath: stepInfo.GetString("attachment-path"),
		DownloadPath:   stepInfo.GetString("download-path"),
		FilenamePath:   stepInfo.GetString("filename-path"),
		AcceptType:     stepInfo.GetString("accept-type"),
		Maximum:        maximum,
		Minimum:        minimum,
		Category:       stepInfo.GetString("category"),
		JSONResult:     stepInfo.GetBool("json-result"),

		Label:                stepInfo.GetString("label"),
		LabelFieldname:       stepInfo.GetString("label-fieldname"),
		Description:          stepInfo.GetString("description"),
		DescriptionFieldname: stepInfo.GetString("description-fieldname"),

		RuleHeight: rules.GetInt("height"),
		RuleWidth:  rules.GetInt("width"),
		RuleTypes:  rules.GetSliceOfString("types"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step UploadAttachments) Name() string {
	return "upload-attachments"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step UploadAttachments) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step UploadAttachments) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step UploadAttachments) RequiredRoles() []string {
	return []string{}
}
