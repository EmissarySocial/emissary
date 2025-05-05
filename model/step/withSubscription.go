package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithProduct is a Step that returns a new Follower Builder
type WithProduct struct {
	SubSteps []Step
}

// NewNewWithProduct returns a fully initialized NewWithProduct object
func NewWithProduct(stepInfo mapof.Any) (WithProduct, error) {

	const location = "NewNewWithProduct"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithProduct{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithProduct{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithProduct) AmStep() {}
