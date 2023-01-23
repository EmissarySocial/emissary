package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithFolder represents an action-step that can update the data.DataMap custom data stored in a Stream
type WithFolder struct {
	SubSteps []Step
}

// NewWithFolder returns a fully initialized WithFolder object
func NewWithFolder(stepInfo mapof.Any) (WithFolder, error) {

	const location = "NewWithFolder"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithFolder{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithFolder{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WithFolder) AmStep() {}
