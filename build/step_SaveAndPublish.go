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

// Get renders this step during a GET request. Implements the Step interface.
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

	// RULE: This step can only run on a Stream builder.
	streamBuilder, ok := builder.(Stream)

	if !ok {
		return Halt().WithError(derp.Internal(location, "Builder must be a StreamBuilder"))
	}

	// Collect Services and Data
	factory := streamBuilder.factory()
	session := streamBuilder.session()
	streamService := factory.Stream()
	stream := streamBuilder._stream

	// Try to load the User from the Database
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByID(session, streamBuilder.AuthenticatedID(), &user); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Loading user", streamBuilder.AuthenticatedID()))
	}

	// Additional rules if this Stream is headed for the user's outbox...
	if step.Outbox {
		// Guarantee this Stream has a context collection.
		if err := streamService.CalcContext(session, stream); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Calculating context for stream", stream))
		}

		// If this Stream is a reply, record it in the local parent's Replies collection.
		if err := streamService.AddReply(session, stream.InReplyTo, stream.ActivityPubURL()); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Adding reply to parent's collection", stream))
		}
	}

	// Try to Publish the Stream to ActivityPub

	// Publish the Stream to the ActivityPub Outbox
	if err := streamService.Publish(session, &user, stream, step.StateID, step.Outbox, step.Republish); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Publishing Stream", stream))
	}

	return nil
}
