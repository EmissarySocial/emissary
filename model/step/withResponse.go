package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithResponse is a Step that returns a new Response Builder
type WithResponse struct {
	SubSteps []Step
}

// NewWithResponse returns a fully initialized WithResponse object
func NewWithResponse(stepInfo mapof.Any) (WithResponse, error) {

	const location = "NewWithResponse"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithResponse{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithResponse{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithResponse) AmStep() {}
