package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/translate"
)

// MakeArchive is a Step that removes a named archive from a Stream
type MakeArchive struct {
	Token       string
	Depth       int
	JSON        bool
	Attachments bool
	Metadata    [][]map[string]any
}

// NewMakeArchive returns a fully initialized MakeArchive object
func NewMakeArchive(stepInfo mapof.Any) (MakeArchive, error) {

	const location = "step.NewMakeArchive"

	metadataAny := stepInfo.GetSliceOfAny("metadata")

	// Convert []any to [][]map[string]any
	metadata := make([][]map[string]any, len(metadataAny))

	for index, item := range metadataAny {
		metadata[index] = convert.SliceOfMap(item)
	}

	// Convert [][]map[string]any to []Pipeline
	if _, err := translate.NewSliceOfPipelines(metadata); err != nil {
		return MakeArchive{}, derp.Wrap(err, location, "Error parsing metadata", stepInfo.GetAny("metadata"), metadata)
	}

	// Return a valid MakeArchive object
	return MakeArchive{
		Token:       stepInfo.GetString("token"),
		Depth:       stepInfo.GetInt("depth"),
		JSON:        stepInfo.GetBool("json"),
		Attachments: stepInfo.GetBool("attachments"),
		Metadata:    metadata,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step MakeArchive) Name() string {
	return "make-archive"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step MakeArchive) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step MakeArchive) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step MakeArchive) RequiredRoles() []string {
	return []string{}
}
