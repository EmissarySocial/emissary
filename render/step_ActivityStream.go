package render

import (
	"io"

	"github.com/benpate/datatype"
)

// StepActivityStream represents an action-step that (will eventually) send ActivityPub updates to subscribers.
type StepActivityStream struct {
	activityType string
}

func NewStepActivityStream(stepInfo datatype.Map) StepActivityStream {

	return StepActivityStream{
		activityType: stepInfo.GetString("type"),
	}
}

// Get displays a form for users to fill out in the browser
func (step StepActivityStream) Get(buffer io.Writer, renderer *Stream) error {
	return nil
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step StepActivityStream) Post(buffer io.Writer, renderer *Stream) error {
	return nil
}
