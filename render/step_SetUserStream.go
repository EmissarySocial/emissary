package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/whisperverse/whisperverse/service"
)

// StepSetUserStream represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepSetUserStream struct {
	service *service.UserStream
	paths   []string
}

func NewStepSetUserStream(userStreamService *service.UserStream, command datatype.Map) StepSetUserStream {

	return StepSetUserStream{
		service: userStreamService,
		paths:   command.GetSliceOfString("paths"),
	}
}

// Get displays a form where users can update stream data
func (step StepSetUserStream) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSetUserStream) Post(buffer io.Writer, renderer Renderer) error {
	/*
		streamRenderer := renderer.(Stream)

		userStream, err := step.service.LoadByUserAndStream(streamRenderer.stream.StreamID, streamRenderer.user.UserID)

		if err != nil {
			return derp.Wrap(err, "render.StepSetUserStream.Post", "Error loading UserStream")
		}
	*/
	return nil
}
