package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithDraft is an action-step that returns a new StreamDraft Builder
type WithDraft struct {
	SubSteps []Step
}

// NewWithDraft returns a fully initialized WithDraft object
func NewWithDraft(stepInfo mapof.Any) (WithDraft, error) {

	const location = "build.NewWithDraft"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithDraft{}, derp.Wrap(err, location, "Invalid 'steps'")
	}

	return WithDraft{
		SubSteps: subSteps,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithDraft) AmStep() {}
