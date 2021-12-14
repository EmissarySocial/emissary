package render

import (
	"io"

	"github.com/benpate/datatype"
)

// StepStreamState represents an action-step that can change a Stream's state
type StepStreamState struct {
	newState string
}

func NewStepStreamState(stepInfo datatype.Map) StepStreamState {

	return StepStreamState{
		newState: stepInfo.GetString("newState"),
	}
}

// Get displays a form for users to fill out in the browser
func (step StepStreamState) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step StepStreamState) Post(buffer io.Writer, renderer Renderer) error {
	streamRenderer := renderer.(Stream)
	streamRenderer.stream.StateID = step.newState
	// TODO: post-change hooks??
	return nil
}
