package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepSetState represents an action-step that can change a Stream's state
type StepSetState struct {
	State string
}

func (step StepSetState) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step StepSetState) Post(builder Builder, _ io.Writer) PipelineBehavior {

	// If the builder is a StateSetter, then try to update the state
	if setter, ok := builder.(StateSetter); ok {

		// This action may still fail (for instance) if the builder wraps
		// a model object that is not a `model.StateSetter`
		if err := setter.setState(step.State); err != nil {
			return Halt().WithError(derp.Wrap(err, "build.stepSetState.Post", "Error setting state"))
		}

		// Success
		return nil
	}

	// Failure (obv)
	return Halt().WithError(derp.NewInternalError("build.stepSetState.Post", "Builder does not implement StateSetter interface"))
}
