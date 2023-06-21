package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepSetState represents an action-step that can change a Stream's state
type StepSetState struct {
	StateID string
}

func (step StepSetState) Get(renderer Renderer, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step StepSetState) Post(renderer Renderer, _ io.Writer) PipelineBehavior {

	// Try to set the state via the Path interface.
	if err := renderer.schema().Set(renderer.object(), "stateId", step.StateID); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.stepSetState.Post", "Error setting stateId", step.StateID))
	}

	return nil
}
