package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithCircle is a Step executes a list of sub-steps on every child of the current Stream
type WithCircle struct {
	SubSteps []Step
}

// NewWithCircle returns a fully initialized WithCircle object
func NewWithCircle(stepInfo mapof.Any) (WithCircle, error) {

	const location = "NewWithCircle"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithCircle{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithCircle{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithCircle) AmStep() {}
