package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepSaveAndPublish is a Step that makes the signed-in User the author of a Stream, moves
// it into a state, and saves it
type StepSaveAndPublish struct {
	StateID   string
	Republish bool
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepSaveAndPublish) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post moves the Stream into the step's state, with the signed-in User as its author
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

	// Move the Stream into its new state.  The save gives a Stream that enters the outbox its
	// context and its place in a local parent's Replies, and queues the task that sends it.
	if err := streamService.Publish(session, &user, stream, step.StateID, step.Republish); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Publishing Stream", stream.StreamID))
	}

	return nil
}
