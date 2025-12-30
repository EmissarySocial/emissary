package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepSaveAndPublish is a Step that can update a stream's PublishDate with the current time.
type StepSaveAndPublish struct {
	StateID   string
	Outbox    bool
	Republish bool
}

func (step StepSaveAndPublish) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepSaveAndPublish) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSaveAndPublish.Post"

	// RULE: Require authentication to publish content
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "User must be authenticated to publish content"))
	}

	streamBuilder, ok := builder.(Stream)

	if !ok {
		return Halt().WithError(derp.Internal(location, "Builder must be a StreamBuilder"))
	}

	factory := streamBuilder.factory()
	stream := streamBuilder._stream

	// Try to load the User from the Database
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByID(builder.session(), streamBuilder.AuthenticatedID(), &user); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to load user", streamBuilder.AuthenticatedID()))
	}

	// Try to Publish the Stream to ActivityPub
	streamService := factory.Stream()

	// Publish the Stream to the ActivityPub Outbox
	if err := streamService.Publish(builder.session(), &user, stream, step.StateID, step.Outbox, step.Republish); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to publish Stream", streamBuilder._stream))
	}

	return nil
}
