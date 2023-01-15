package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/maps"
)

// WithFollower represents an action-step that can update the data.DataMap custom data stored in a Stream
type WithFollower struct {
	SubSteps []Step
}

// NewWithFollower returns a fully initialized WithFollower object
func NewWithFollower(stepInfo maps.Map) (WithFollower, error) {

	const location = "NewWithFollower"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithFollower{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithFollower{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WithFollower) AmStep() {}
