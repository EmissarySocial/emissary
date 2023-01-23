package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithBlock represents an action-step that can update the data.DataMap custom data stored in a Stream
type WithBlock struct {
	SubSteps []Step
}

// NewWithBlock returns a fully initialized WithBlock object
func NewWithBlock(stepInfo mapof.Any) (WithBlock, error) {

	const location = "NewWithBlock"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithBlock{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithBlock{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WithBlock) AmStep() {}
