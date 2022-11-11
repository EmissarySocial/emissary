package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/maps"
)

// WithInboxFolder represents an action-step that can update the data.DataMap custom data stored in a Stream
type WithInboxFolder struct {
	SubSteps []Step
}

// NewWithInboxFolder returns a fully initialized WithInboxFolder object
func NewWithInboxFolder(stepInfo maps.Map) (WithInboxFolder, error) {

	const location = "NewWithInboxFolder"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithInboxFolder{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithInboxFolder{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WithInboxFolder) AmStep() {}
