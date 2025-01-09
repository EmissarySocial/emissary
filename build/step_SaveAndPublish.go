package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepSaveAndPublish is a Step that can update a stream's PublishDate with the current time.
type StepSaveAndPublish struct {
	Outbox bool
}

func (step StepSaveAndPublish) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepSaveAndPublish) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSaveAndPublish.Post"

	streamBuilder := builder.(Stream)
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

	if err := streamService.Publish(&user, streamBuilder._stream, step.Outbox); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error publishing stream", streamBuilder._stream))
	}

	return nil
}
