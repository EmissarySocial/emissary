package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithAttachment is a Step that returns a new Attachment Builder
type WithAttachment struct {
	SubSteps []Step
}

// NewWithAttachment returns a fully initialized WithAttachment object
func NewWithAttachment(stepInfo mapof.Any) (WithAttachment, error) {

	const location = "NewWithAttachment"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithAttachment{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithAttachment{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithAttachment) AmStep() {}
