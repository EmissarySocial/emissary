package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Publisher struct {
	streamService   *Stream
	followerService *Follower
	userService     *User
}

func NewPublisher(streamService *Stream, followerService *Follower, userService *User) Publisher {
	return Publisher{
		streamService:   streamService,
		followerService: followerService,
		userService:     userService,
	}
}

func (publisher Publisher) Publish(stream *model.Stream, userID primitive.ObjectID) error {

	// RULE: Update the stream (if necessary)
	if err := publisher.setPublishedData(stream, userID); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error setting published data", stream.ID)
	}

	// RULE: Send notifications (if necessary)
	if err := publisher.notifyFollowers(stream); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error sending notifications", stream)
	}

	return nil
}

// setPublishData marks this stream as "published"
func (publisher Publisher) setPublishedData(stream *model.Stream, userID primitive.ObjectID) error {

	// IF THIS STREAM HAS NOT ALREADY BEEN PUBLISHED...
	if stream.PublishDate == 0 {

		// RULE: Set the publish date to the current time.
		stream.PublishDate = time.Now().Unix()

		// RULE: Set Author to the currently logged in user.
		user := model.NewUser()
		if err := publisher.userService.LoadByID(userID, &user); err != nil {
			return derp.Wrap(err, "service.Publisher.Publish", "Error loading user", userID)
		}

		stream.Document.Author = user.PersonLink()

		// Re-save the Stream with the updated values.
		if err := publisher.streamService.Save(stream, "Publish"); err != nil {
			return derp.Wrap(err, "render.StepPublish", "Error saving stream", stream)
		}
	}

	return nil
}

// notifyFollowers creates an "outbox-item" `Stream` and sends
// notifications to all followers of the stream's author
func (publisher Publisher) notifyFollowers(stream *model.Stream) error {

	// Try to load an existing outbox item for this stream
	outboxItem := model.NewStream()
	err := publisher.streamService.LoadByOriginID(stream.StreamID, &outboxItem)

	// No Error means that we already have an outbox item for this stream.
	if err == nil {

		// TODO: CRITICAL: Send UPDATE notifications to all internal followers.

		// TODO: CRITICAL: Send UPDATE notifications to all external followers.
		return nil
	}

	// "Not Found" error means that this is the first time we're sending notifications.
	if derp.NotFound(err) {

		// Get a new outbox-item for this stream
		outboxItem := stream.OutboxItem()

		// Save it to the database.
		if err := publisher.streamService.Save(&outboxItem, "Publish"); err != nil {
			return derp.Wrap(err, "service.Publisher.Publish", "Error saving outbox item", stream)
		}

		// TODO: CRITICAL: Send CREATE notifications to all internal followers.

		// TODO: CRITICAL: Send CREATE notifications to all external followers.

	}

	// Fall through to here means that it's a legitimate error, so let's
	// just shut that whole thing down.
	return derp.Wrap(err, "service.Publisher.Publish", "Error loading outbox item", stream.StreamID)
}
