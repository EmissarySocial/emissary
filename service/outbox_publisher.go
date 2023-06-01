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
func (service Outbox) Publish(userID primitive.ObjectID, url string, activity mapof.Any) error {

	// Write a new OutboxMessage to the database
	outboxMessage := model.NewOutboxMessage()
	outboxMessage.UserID = userID
	outboxMessage.URL = url

	if err := service.Save(&outboxMessage, "Updated"); err != nil {
		return derp.Wrap(err, "service.Outbox.NewPublish", "Error saving outbox message", outboxMessage)
	}

	// Send notifications to followers on all push channels
	activityPubFollowers, webSubFollowers := service.followerService.FollowerChannels(userID)

	go service.SendNotifications_ActivityPub(userID, activityPubFollowers, activity)
	go service.SendNotifications_WebSub(webSubFollowers, activity)

	// Success!!
	return nil
}

// UnPublish deletes an OutboxMessage from the Outbox, and sends notifications to all followers
func (service Outbox) UnPublish(userID primitive.ObjectID, url string, activity mapof.Any) error {

	const location = "service.Outbox.Unpublish"

	// Try to load the existing outbox message
	outboxMessage := model.NewOutboxMessage()
	if err := service.LoadByURL(userID, url, &outboxMessage); err != nil {
		if derp.NotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Error loading outbox message", userID, url)
	}

	// Fall through means we have a valid outboxMessage to unpublish.
	if err := service.Delete(&outboxMessage, "Un-Publishing"); err != nil {
		return derp.Wrap(err, location, "Error deleting outbox message", outboxMessage)
	}

	// Send notifications to followers on all push channels
	activityPubFollowers, webSubFollowers := service.followerService.FollowerChannels(userID)

	go service.SendNotifications_ActivityPub(userID, activityPubFollowers, activity)
	go service.drainChannel(webSubFollowers)

	// Hey-oh!
	return nil
}

func (service Outbox) SendNotifications_ActivityPub(userID primitive.ObjectID, followers <-chan model.Follower, activity mapof.Any) error {

	const location = "service.Outbox.SendNotifications_ActivityPub"

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
func (service Outbox) SendNotifications_WebSub(followers <-chan model.Follower, activity mapof.Any) error {

	// Queue up all ActivityPub messages to be sent
	for follower := range followers {
		service.queue.Run(NewTaskSendWebSubMessage(follower))
	}

	return nil
}

func (service Outbox) drainChannel(channel <-chan model.Follower) {
	for range channel {
	}
}
