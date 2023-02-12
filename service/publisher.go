package service

import (
	"math"
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

	// NOTE: It's okay to re-publish multiple times.
	isPublished := stream.IsPublished()

	// Get the current User record
	user := model.NewUser()
	if err := publisher.userService.LoadByID(userID, &user); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading user", userID)
	}

	// RULE: Update the stream (if necessary)
	if err := publisher.setPublishedData(stream, &user); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error setting published data", stream.ID)
	}

	// RULE: Send notifications (if necessary)
	if err := publisher.notifyFollowers(stream); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error sending notifications", stream)
	}

	// RULE: Send ActivityPub Create/Update messages to federated peers
	if isPublished {
		if err := publisher.sendActivityPub_Create(stream, &user); err != nil {
			return derp.Wrap(err, "service.Publisher.Publish", "Error sending ActivityPub messages", stream)
		}
	} else {
		if err := publisher.sendActivityPub_Update(stream, &user); err != nil {
			return derp.Wrap(err, "service.Publisher.Publish", "Error sending ActivityPub messages", stream)
		}
	}

	return nil
}

func (publisher Publisher) Unpublish(stream *model.Stream, userID primitive.ObjectID) error {

	// RULE: Set the "UnPublish" date
	stream.UnPublishDate = time.Now().Unix()
	if err := publisher.streamService.Save(stream, "Un-Publish"); err != nil {
		return derp.Wrap(err, "render.StepPublish", "Error saving stream", stream)
	}

	// Get the current User record
	user := model.NewUser()
	if err := publisher.userService.LoadByID(userID, &user); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading user", userID)
	}

	// RULE: Send ActivityPub Delete messages to federated peers
	if err := publisher.sendActivityPub_Delete(stream, &user); err != nil {
		return derp.Wrap(err, "service.Publisher.Unpublish", "Error sending ActivityPub messages", stream)
	}

	// Hey-oh!
	return nil
}

// setPublishData marks this stream as "published"
func (publisher Publisher) setPublishedData(stream *model.Stream, user *model.User) error {

	// RULE: IF this stream is not yet published, then set the publish date
	if stream.PublishDate > time.Now().Unix() {
		stream.PublishDate = time.Now().Unix()
	}

	// RULE: Move unpublish date all the way to the end of time.
	// TODO: LOW: May want to set automatic unpublish dates later...
	stream.UnPublishDate = math.MaxInt64

	// RULE: Set Author to the currently logged in user.
	stream.Document.Author = user.PersonLink()

	// Re-save the Stream with the updated values.
	if err := publisher.streamService.Save(stream, "Publish"); err != nil {
		return derp.Wrap(err, "render.StepPublish", "Error saving stream", stream)
	}

	// Done.
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

// sendActivityPub_Create sends a "Create" message to the outbox of the given user
func (publisher Publisher) sendActivityPub_Create(stream *model.Stream, user *model.User) error {
	// TODO: CRITICAL: Send ActivityPub Create message to the outbox of all followers
	return nil
}

// sendActivityPub_Update sends an "Update" message to the outbox of the given user
func (publisher Publisher) sendActivityPub_Update(stream *model.Stream, user *model.User) error {
	// TODO: CRITICAL: Send ActivityPub Update message to the outbox of all followers
	return nil
}

// sendActivityPub_Delete sends a "Delete" message to the outbox of the given user
func (publisher Publisher) sendActivityPub_Delete(stream *model.Stream, user *model.User) error {
	// TODO: CRITICAL: Send ActivityPub Delete message to the outbox of all followers
	return nil
}
