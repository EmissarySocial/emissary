package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithNextSibling is a Step that returns a new Stream Builder keyed to the next sibling of the current Stream
type WithNextSibling struct {
	SubSteps []Step
}

// NewWithNextSibling returns a fully initialized WithNextSibling object
func NewWithNextSibling(stepInfo mapof.Any) (WithNextSibling, error) {

	const location = "NewWithNextSibling"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithNextSibling{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithNextSibling{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithNextSibling) AmStep() {}
