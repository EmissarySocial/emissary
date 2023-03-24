package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepUnPublish represents an action-step that can update a stream's PublishDate with the current time.
type StepUnPublish struct {
	Role string
}

func (step StepUnPublish) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepUnPublish) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepUnPublish) Post(renderer Renderer) error {

	const location = "render.StepUnPublish.Post"

	// Require that the user is signed in to perform this action
	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError(location, "User is not authenticated", nil)
	}

	// Use the publisher service to execute publishing rules
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream

	outboxService := renderer.factory().Outbox()
	outboxService.Unpublish(renderer.AuthenticatedID(), stream)

	// TODO: MEDIUM: Do we NEED to 'Tombstone' records when the stream has been deleted?

	return nil
}
