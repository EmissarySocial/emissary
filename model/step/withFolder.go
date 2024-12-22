package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithFolder is a Step that returns a new Folder Builder
type WithFolder struct {
	SubSteps []Step
}

// NewWithFolder returns a fully initialized WithFolder object
func NewWithFolder(stepInfo mapof.Any) (WithFolder, error) {

	const location = "model.step.NewWithFolder"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithFolder{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithFolder{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithFolder) AmStep() {}
