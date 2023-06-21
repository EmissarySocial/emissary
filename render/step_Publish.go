package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepPublish represents an action-step that can update a stream's PublishDate with the current time.
type StepPublish struct{}

func (step StepPublish) Get(renderer Renderer, _ io.Writer) ExitCondition {
	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepPublish) Post(renderer Renderer, _ io.Writer) ExitCondition {

	const location = "render.StepPublish.Post"

	// Require that the user is signed in to perform this action
	if !renderer.IsAuthenticated() {
		return ExitError(derp.NewUnauthorizedError(location, "User is not authenticated", nil))
	}

	streamRenderer := renderer.(*Stream)
	factory := streamRenderer.factory()

	// Try to load the User from the Database
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByID(streamRenderer.AuthenticatedID(), &user); err != nil {
		return ExitError(derp.Wrap(err, location, "Error loading user", streamRenderer.AuthenticatedID()))
	}

	// Try to Publish the Stream to ActivityPub
	streamService := factory.Stream()

	if err := streamService.Publish(&user, streamRenderer.stream); err != nil {
		return ExitError(derp.Wrap(err, location, "Error publishing stream", streamRenderer.stream))
	}

	return nil
}
