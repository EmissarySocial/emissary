package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithChildren represents an action-step that can update the data.DataMap custom data stored in a Stream
type WithChildren struct {
	SubSteps []Step
}

// NewWithChildren returns a fully initialized WithChildren object
func NewWithChildren(stepInfo mapof.Any) (WithChildren, error) {

	const location = "NewWithChildren"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithChildren{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithChildren{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WithChildren) AmStep() {}
