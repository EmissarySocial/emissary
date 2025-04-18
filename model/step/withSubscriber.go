package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithSubscriber is a Step that returns a new Follower Builder
type WithSubscriber struct {
	SubSteps []Step
}

// NewWithSubscriber returns a fully initialized WithSubscriber object
func NewWithSubscriber(stepInfo mapof.Any) (WithSubscriber, error) {

	const location = "NewWithSubscriber"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithSubscriber{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithSubscriber{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithSubscriber) AmStep() {}
