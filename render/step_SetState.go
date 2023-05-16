package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepSetState represents an action-step that can change a Stream's state
type StepSetState struct {
	StateID string
}

func (step StepSetState) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepSetState) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step StepSetState) Post(renderer Renderer, _ io.Writer) error {

	// Try to set the state via the Path interface.
	object := renderer.object()
	if err := renderer.schema().Set(object, "stateId", step.StateID); err != nil {
		return derp.Wrap(err, "render.stepSetState.Post", "Error setting stateId", step.StateID)
	}

	return nil
}
