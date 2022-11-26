package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepPublish represents an action-step that can update a stream's PublishDate with the current time.
type StepPublish struct {
	Role string
}

func (step StepPublish) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepPublish) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepPublish) Post(renderer Renderer) error {

	// Require that the user is signed in to perform this action
	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError("render.StepPublish", "User is not authenticated", nil)
	}

	// Use the publisher service to execute publishing rules
	streamRenderer := renderer.(Stream)
	stream := streamRenderer.stream

	publisherService := renderer.factory().Publisher()
	publisherService.Publish(stream, renderer.AuthenticatedID())

	return nil
}
