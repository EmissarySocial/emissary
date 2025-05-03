package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepSaveAndPublish is a Step that can update a stream's PublishDate with the current time.
type StepSaveAndPublish struct {
	Outbox    bool
	Republish bool
}

func (step StepSaveAndPublish) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepSaveAndPublish) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSaveAndPublish.Post"

	streamBuilder, ok := builder.(Stream)

	if !ok {
		return Halt().WithError(derp.NewInternalError(location, "Builder must be a StreamBuilder"))
	}

	factory := streamBuilder.factory()
	stream := streamBuilder._stream

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
	geocoder := factory.Geocode()

	// Try to geocode any Places in this Stream. If there are Geocoder errors,
	// then a task will be queued to retry the geocode in 30 seconds.
	if err := geocoder.GeocodeAndQueue(stream); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error geocoding stream", streamBuilder._stream))
	}

	// Publish the Stream to the ActivityPub Outbox
	if err := streamService.Publish(&user, stream, step.Outbox, step.Republish); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error publishing stream", streamBuilder._stream))
	}

	return nil
}
