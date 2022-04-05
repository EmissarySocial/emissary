package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/path"
)

// StepStreamState represents an action-step that can change a Stream's state
type StepStreamState struct {
	stateID string

	BaseStep
}

func NewStepStreamState(stepInfo datatype.Map) (StepStreamState, error) {

	return StepStreamState{
		stateID: stepInfo.GetString("state"),
	}, nil
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step StepStreamState) Post(_ Factory, renderer Renderer, _ io.Writer) error {

	// Try to set the state via the Path interface.
	return path.Set(renderer.object(), "stateId", step.stateID)
}
