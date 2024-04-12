package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithFollower is an action-step that returns a new Follower Builder
type WithFollower struct {
	SubSteps []Step
}

// NewWithFollower returns a fully initialized WithFollower object
func NewWithFollower(stepInfo mapof.Any) (WithFollower, error) {

	const location = "NewWithFollower"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithFollower{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithFollower{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithFollower) AmStep() {}
