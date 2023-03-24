package service

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Publish/Unpublish Methods
 ******************************************/

func (service Outbox) Publish(userID primitive.ObjectID, stream *model.Stream, objectType string) error {

	// Get the current User record
	user := model.NewUser()
	if err := service.userService.LoadByID(userID, &user); err != nil {
		return derp.Wrap(err, "service.Outbox.Publish", "Error loading user", userID)
	}

	// TODO: LOW: Hack-y way to set the "Published" date.  DO BETTER.
	stream.Document.Type = objectType

	// Build the outbox message
	outboxMessage := model.NewOutboxMessage()
	outboxMessage.UserID = user.UserID
	outboxMessage.ObjectType = "Stream"
	outboxMessage.ObjectID = stream.StreamID
	outboxMessage.ParentID = stream.ParentID
	outboxMessage.Activity = service.makeActivity(&user, stream)

	// Save the outbox message
	if err := service.Save(&outboxMessage, "Created"); err != nil {
		return derp.Wrap(err, "service.Outbox.Publish", "Error saving outbox message", outboxMessage)
	}

	// RULE: Save changes to the Stream
	if err := service.setPublished(stream, &user); err != nil {
		return derp.Wrap(err, "service.Outbox.Publish", "Error setting published data", stream.ID)
	}

	// RULE: Send ActivityPub Nofitications to all followers
	if err := service.SendNotifications_ActivityPub(outboxMessage); err != nil {
		return derp.Wrap(err, "service.Outbox.Publish", "Error sending ActivityPub notifications", stream)
	}

	// RULE: Send WebSub Nofitications to all followers
	if err := service.SendNotifications_WebSub(outboxMessage); err != nil {
		return derp.Wrap(err, "service.Outbox.Publish", "Error sending ActivityPub notifications", stream)
	}

	// Success!
	return nil
}

func (service Outbox) Unpublish(userID primitive.ObjectID, stream *model.Stream) error {

	const location = "service.Outbox.Unpublish"

	// TODO: Delete all outbox items for unpublished streams.
	messagesToDelete, err := service.QueryByObject("Stream", stream.StreamID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading outbox Messages", stream)
	}

	for _, message := range messagesToDelete {
		if err := service.Delete(&message, "Un-Publish"); err != nil {
			return derp.Wrap(err, location, "Error deleting outbox message", message)
		}
	}

	// RULE: Set the "UnPublish" date
	stream.UnPublishDate = time.Now().Unix()
	if err := service.streamService.Save(stream, "Un-Publish"); err != nil {
		return derp.Wrap(err, location, "Error updating stream", stream)
	}

	// Get the current User record
	user := model.NewUser()
	if err := service.userService.LoadByID(userID, &user); err != nil {
		return derp.Wrap(err, location, "Error loading user", userID)
	}

	// Build the outbox message
	outboxMessage := model.NewOutboxMessage()
	outboxMessage.UserID = user.UserID
	outboxMessage.ObjectType = "Stream"
	outboxMessage.ObjectID = stream.StreamID
	outboxMessage.ParentID = stream.ParentID
	outboxMessage.Activity = service.makeActivity(&user, stream)

	// RULE: Send ActivityPub Delete messages to federated peers
	if err := service.SendNotifications_ActivityPub(outboxMessage); err != nil {
		return derp.Wrap(err, location, "Error sending ActivityPub messages", stream)
	}

	// Hey-oh!
	return nil
}

func (service Outbox) SendNotifications_ActivityPub(message model.OutboxMessage) error {

	const location = "service.Outbox.SendNotifications_ActivityPub"

	// Get the iterator of followers to notify
	followers, err := service.followerService.ChannelActivityPub(message.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading followers", message)
	}

	// If the channel is nil, then there are no followers to notify, so we're done.
	if followers == nil {
		return nil
	}

	// Load the ActivityPub Actor for this Stream
	actor, err := service.userService.ActivityPubActor(message.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading actor", message)
	}

	// Queue up all ActivityPub messages to be sent
	for follower := range followers {
		service.queue.Run(pub.SendQueueTask(actor, message.Activity, follower.Actor.ProfileURL))
	}

	return nil
}

// TODO: HIGH: Thoroughly re-test WebSub notifications.  They've been rebuilt from scratch.
func (service Outbox) SendNotifications_WebSub(message model.OutboxMessage) error {

	const location = "SendNotifications_WebSub"

	if message.ObjectType != "Stream" {
		return nil
	}

	// Get the iterator of followers to notify
	followers, err := service.followerService.ChannelWebSub(message.ObjectID, message.ParentID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading followers", message)
	}

	// If the channel is nil, then there are no followers to notify, so we're done.
	if followers == nil {
		return nil
	}

	// Queue up all ActivityPub messages to be sent
	for follower := range followers {
		service.queue.Run(NewTaskSendWebSubMessage(message, follower))
	}

	return nil
}

/******************************************
 * Helper Methods
 ******************************************/

// setPublished marks this stream as "published"
func (service Outbox) setPublished(stream *model.Stream, user *model.User) error {

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
	if err := service.streamService.Save(stream, "Publish"); err != nil {
		return derp.Wrap(err, "service.Outbox.setPublished", "Error saving stream", stream)
	}

	// TODO: CRITICAL: Send Delete activities when unpublishing a stream

	// Done.
	return nil
}

func (service Outbox) makeActivity(user *model.User, stream *model.Stream) mapof.Any {

	return mapof.Any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       service.streamService.ActivityPubID(stream),
		"actor":    user.ActivityPubURL(),
		"type":     service.guessActivityType(stream),
		"object":   stream.GetJSONLD(),
	}
}

func (service Outbox) guessActivityType(stream *model.Stream) string {

	if stream.Journal.DeleteDate > 0 {
		return vocab.ActivityTypeDelete
	}

	if stream.PublishDate > time.Now().Unix() {
		return vocab.ActivityTypeCreate
	}

	return vocab.ActivityTypeUpdate
}
