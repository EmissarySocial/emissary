package render

import (
	"io"
	"time"

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

	factory := renderer.factory()
	activityPub := factory.ActivityPub()

	actor, err := activityPub.LoadActor(renderer.AuthenticatedID())

	streamRenderer := renderer.(*Stream)
	streamRenderer.stream.PublishDate = time.Now().UnixMilli()
	return nil
}
