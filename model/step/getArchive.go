package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/translate"
)

// GetArchive is a Step that removes a named archive from a Stream
type GetArchive struct {
	Token       string
	Depth       int
	JSON        bool
	Attachments bool
	Metadata    [][]map[string]any
}

// NewGetArchive returns a fully initialized GetArchive object
func NewGetArchive(stepInfo mapof.Any) (GetArchive, error) {

	const location = "step.NewGetArchive"

	metadataAny := stepInfo.GetSliceOfAny("metadata")

	// Convert []any to [][]map[string]any
	metadata := make([][]map[string]any, len(metadataAny))

	for index, item := range metadataAny {
		metadata[index] = convert.SliceOfMap(item)
	}

	// Convert [][]map[string]any to []Pipeline
	if _, err := translate.NewSliceOfPipelines(metadata); err != nil {
		return GetArchive{}, derp.Wrap(err, location, "Error parsing metadata", stepInfo.GetAny("metadata"), metadata)
	}

	// Return a valid MakeArchive object
	return GetArchive{
		Token:       stepInfo.GetString("token"),
		Depth:       stepInfo.GetInt("depth"),
		JSON:        stepInfo.GetBool("json"),
		Attachments: stepInfo.GetBool("attachments"),
		Metadata:    metadata,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step GetArchive) Name() string {
	return "get-archive"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step GetArchive) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step GetArchive) RequiredRoles() []string {
	return []string{}
}
