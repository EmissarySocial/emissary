package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithPrevSibling represents an action-step that can update the data.DataMap custom data stored in a Stream
type WithPrevSibling struct {
	SubSteps []Step
}

// NewWithPrevSibling returns a fully initialized WithPrevSibling object
func NewWithPrevSibling(stepInfo mapof.Any) (WithPrevSibling, error) {

	const location = "NewWithPrevSibling"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithPrevSibling{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithPrevSibling{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WithPrevSibling) AmStep() {}
