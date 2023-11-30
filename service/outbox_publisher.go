package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"willnorris.com/go/webmention"
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
	go service.sendNotifications_ActivityPub(userID, activity)
	go service.sendNotifications_WebSub(userID, activity)
	go service.sendNotifications_WebMention(activity)

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

	// Send notifications to ActivityPub followers (WebSub does not have an Undo)
	go service.sendNotifications_ActivityPub(userID, activity)

	// Hey-oh!
	return nil
}

/******************************************
 * Internal Publishing Methods
 ******************************************/

// sendNotifications_ActivityPub sends ActivityPub updates to all Followers
func (service Outbox) sendNotifications_ActivityPub(userID primitive.ObjectID, activity mapof.Any) {

	const location = "service.Outbox.sendNotifications_ActivityPub"

	// Load the ActivityPub Actor (with Followers)
	actor, err := service.userService.ActivityPubActor(userID, true)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading actor", userID))
		return
	}

	// Use the Actor to send the Activity to all recipients
	actor.Send(activity)
}

// TODO: HIGH: Thoroughly re-test WebSub notifications.  They've been rebuilt from scratch.
func (service Outbox) sendNotifications_WebSub(userID primitive.ObjectID, activity mapof.Any) {

	const location = "service.Outbox.sendNotifications_WebSub"

	// Get this User's Followers from the database
	followers, err := service.followerService.WebSubFollowersChannel(userID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading Followers", userID))
	}

	// Queue up all ActivityPub messages to be sent
	for follower := range followers {
		service.queue.Push(NewTaskSendWebSubMessage(follower))
	}
}

// sendNotifications_WebMention sends WebMention updates to external websites that are
// mentioned in this stream.  This is here (and not in the outbox service)
// because we need to render the content in order to discover outbound links.
func (service Outbox) sendNotifications_WebMention(activity mapof.Any) {

	// Locate the object ID for this acticity
	object := activity.GetMap(vocab.PropertyObject)
	id := object.GetString(vocab.PropertyID)
	content := activity.GetString(vocab.PropertyContent)

	// Discover all webmention links in the content
	reader := strings.NewReader(content)
	links, err := webmention.DiscoverLinksFromReader(reader, id, "")

	if err != nil {
		derp.Report(derp.Wrap(err, "mention.SendWebMention.Run", "Error discovering webmention links", activity))
		return
	}

	// If no links, peace out, homie.
	if len(links) == 0 {
		return
	}

	// Add background tasks to TRY sending webmentions to every link we found
	for _, link := range links {
		service.queue.Push(NewTaskSendWebMention(id, link))
	}
}
