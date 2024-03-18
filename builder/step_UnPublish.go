package builder

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepUnPublish represents an action-step that can update a stream's PublishDate with the current time.
type StepUnPublish struct {
	Role string
}

func (step StepUnPublish) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepUnPublish) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepUnPublish.Post"

	streamBuilder := builder.(*Stream)
	factory := streamBuilder.factory()

	// Try to load the User from the Database
	userService := factory.User()
	user := model.NewUser()

	if builder.IsAuthenticated() {
		if err := userService.LoadByID(streamBuilder.AuthenticatedID(), &user); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error loading user", streamBuilder.AuthenticatedID()))
		}
	}

	// Try to Publish the Stream to ActivityPub
	streamService := factory.Stream()

	if err := streamService.UnPublish(&user, streamBuilder._stream); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error publishing stream", streamBuilder._stream))
	}

	return nil
}
