package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepSetState represents an action-step that can change a Stream's state
type StepSetState struct {
	State string
}

func (step StepSetState) Get(renderer Renderer, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step StepSetState) Post(renderer Renderer, _ io.Writer) PipelineBehavior {

	// If the renderer is a StateSetter, then try to update the state
	if setter, ok := renderer.(StateSetter); ok {

		// This action may still fail (for instance) if the renderer wraps
		// a model object that is not a `model.StateSetter`
		if err := setter.setState(step.State); err != nil {
			return Halt().WithError(derp.Wrap(err, "render.stepSetState.Post", "Error setting state"))
		}

		// Success
		return nil
	}

	// Failure (obv)
	return Halt().WithError(derp.NewInternalError("render.stepSetState.Post", "Renderer does not implement StateSetter interface"))
}
