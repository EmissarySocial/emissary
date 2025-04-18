package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithSubscription is a Step that returns a new Follower Builder
type WithSubscription struct {
	SubSteps []Step
}

// NewNewWithSubscription returns a fully initialized NewWithSubscription object
func NewWithSubscription(stepInfo mapof.Any) (WithSubscription, error) {

	const location = "NewNewWithSubscription"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithSubscription{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithSubscription{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithSubscription) AmStep() {}
