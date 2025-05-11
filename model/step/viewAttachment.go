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

// AmStep is here only to verify that this struct is a build pipeline step
func (step ViewAttachment) AmStep() {}
