package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithRule represents an action-step that can update the data.DataMap custom data stored in a Stream
type WithRule struct {
	SubSteps []Step
}

// NewWithRule returns a fully initialized WithRule object
func NewWithRule(stepInfo mapof.Any) (WithRule, error) {

	const location = "NewWithRule"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithRule{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithRule{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WithRule) AmStep() {}
