package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/path"
)

// StepStreamState represents an action-step that can change a Stream's state
type StepStreamState struct {
	stateID string
}

func NewStepStreamState(stepInfo datatype.Map) StepStreamState {

	return StepStreamState{
		stateID: stepInfo.GetString("state"),
	}
}

// Get displays a form for users to fill out in the browser
func (step StepStreamState) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step StepStreamState) Post(buffer io.Writer, renderer Renderer) error {

	// Try to set the state via the Path interface.
	return path.Set(renderer.object(), "stateId", step.stateID)
}
