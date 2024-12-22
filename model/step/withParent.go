package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithParent is a Step that returns a new Stream Builder keyed to the parent of the current Stream
type WithParent struct {
	SubSteps []Step
}

// NewWithParent returns a fully initialized WithParent object
func NewWithParent(stepInfo mapof.Any) (WithParent, error) {

	const location = "build.NewWithParent"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithParent{}, derp.Wrap(err, location, "Invalid 'steps'")
	}

	return WithParent{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithParent) AmStep() {}
