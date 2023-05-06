package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Publish/Unpublish Methods
 ******************************************/

// Publish adds/updates an OutboxMessage in the Outbox, and sends notifications to all followers
func (service Outbox) Publish(userID primitive.ObjectID, objectID primitive.ObjectID, activity mapof.Any) error {

	// Load or Create an OutboxMessage
	outboxMessage, err := service.LoadOrCreate(userID, objectID)

	if err != nil {
		return derp.Wrap(err, "service.Outbox.NewPublish", "Error loading outbox message", userID, objectID)
	}

	// Assign the activity to the outbox message
	outboxMessage.Activity = activity

	if err := service.Save(&outboxMessage, "Updated"); err != nil {
		return derp.Wrap(err, "service.Outbox.NewPublish", "Error saving outbox message", outboxMessage)
	}

	// RULE: Send ActivityPub Nofitications to all followers
	if err := service.SendNotifications_ActivityPub(userID, activity); err != nil {
		return derp.Wrap(err, "service.Outbox.Publish", "Error sending ActivityPub notifications", outboxMessage)
	}

	// RULE: Send WebSub Nofitications to all followers
	if err := service.SendNotifications_WebSub(outboxMessage); err != nil {
		return derp.Wrap(err, "service.Outbox.Publish", "Error sending ActivityPub notifications", outboxMessage)
	}

	// Success!!
	return nil
}

// UnPublish deletes an OutboxMessage from the Outbox, and sends notifications to all followers
func (service Outbox) UnPublish(userID primitive.ObjectID, objectID primitive.ObjectID, activity mapof.Any) error {

	const location = "service.Outbox.Unpublish"

	// Try to load the existing outbox message
	outboxMessage := model.NewOutboxMessage()
	if err := service.LoadByObjectID(userID, objectID, &outboxMessage); err != nil {
		return derp.Wrap(err, location, "Error loading outbox message", userID, objectID)
	}

	// Fall through means we have a valid outboxMessage to unpublish.
	if err := service.Delete(&outboxMessage, "Un-Publishing"); err != nil {
		return derp.Wrap(err, location, "Error deleting outbox message", outboxMessage)
	}

	if err := service.SendNotifications_ActivityPub(userID, activity); err != nil {
		return derp.Wrap(err, location, "Error sending ActivityPub notifications", outboxMessage)
	}

	// Hey-oh!
	return nil
}

func (service Outbox) SendNotifications_ActivityPub(userID primitive.ObjectID, activity mapof.Any) error {

	const location = "service.Outbox.SendNotifications_ActivityPub"

	// Get the iterator of followers to notify
	followers, err := service.followerService.ChannelActivityPub(userID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading followers", userID)
	}

	// If the channel is nil, then there are no followers to notify, so we're done.
	if followers == nil {
		return nil
	}

	// Load the ActivityPub Actor for this Stream
	actor, err := service.userService.ActivityPubActor(userID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading actor", userID)
	}

	// Queue up all ActivityPub messages to be sent
	for follower := range followers {
		service.queue.Run(pub.SendQueueTask(actor, activity, follower.Actor.ProfileURL))
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
