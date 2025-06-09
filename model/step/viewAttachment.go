package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/rosetta/translate"
)

// ViewAttachment is a Step that can render a Stream into a CSS stylesheet
type ViewAttachment struct {
	Categories sliceof.String     // Attachments must match one of these Categories to be accessible
	Formats    sliceof.String     // Allowed file types (e.g., "pdf", "docx")
	Widths     sliceof.Int        // The width(s) of the attachment (if image or video)
	Heights    sliceof.Int        // The height(s) of the attachment (if image or video)
	Bitrates   sliceof.Int        // The bitrate(s) of the attachment (if audio or video)
	Metadata   translate.Pipeline // Mapping to use when generating metadata
}

// NewViewAttachment generates a fully initialized ViewAttachment step.
func NewViewAttachment(stepInfo mapof.Any) (ViewAttachment, error) {

	const location = "build.NewViewAttachment"

	// Validate the Type(s)
	formats := stepInfo.GetSliceOfString("format")

	if len(formats) == 0 {
		return ViewAttachment{}, derp.InternalError(location, "At least one format is required")
	}

	// Compile the metadata pipeline
	metadata, err := translate.NewFromMap(stepInfo.GetSliceOfPlainMap("metadata")...)

	if err != nil {
		return ViewAttachment{}, derp.Wrap(err, location, "Error parsing metadata pipeline")
	}

	// Return the new step
	return ViewAttachment{
		Categories: stepInfo.GetSliceOfString("category"),
		Formats:    formats,
		Heights:    stepInfo.GetSliceOfInt("height"),
		Widths:     stepInfo.GetSliceOfInt("width"),
		Bitrates:   stepInfo.GetSliceOfInt("bitrate"),
		Metadata:   metadata,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ViewAttachment) Name() string {
	return "view-attachment"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ViewAttachment) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ViewAttachment) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ViewAttachment) RequiredRoles() []string {
	return []string{}
}
