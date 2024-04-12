package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithMessage is an action-step that returns a new Message Builder
type WithMessage struct {
	SubSteps []Step
}

// NewWithMessage returns a fully initialized WithMessage object
func NewWithMessage(stepInfo mapof.Any) (WithMessage, error) {

	const location = "NewWithMessage"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithMessage{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithMessage{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithMessage) AmStep() {}
