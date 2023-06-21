package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepUnPublish represents an action-step that can update a stream's PublishDate with the current time.
type StepUnPublish struct {
	Role string
}

func (step StepUnPublish) Get(renderer Renderer, _ io.Writer) ExitCondition {
	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepUnPublish) Post(renderer Renderer, _ io.Writer) ExitCondition {

	const location = "render.StepUnPublish.Post"

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

	if err := streamService.UnPublish(&user, streamRenderer.stream); err != nil {
		return ExitError(derp.Wrap(err, location, "Error publishing stream", streamRenderer.stream))
	}

	return nil
}
