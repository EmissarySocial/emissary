package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/translate"
)

// MakeArchive represents an action-step that removes a named archive from a Stream
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

// AmStep is here only to verify that this struct is a build pipeline step
func (step MakeArchive) AmStep() {}
