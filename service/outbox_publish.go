package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"willnorris.com/go/webmention"
)

/******************************************
 * Publish/Unpublish Methods
 ******************************************/

// Publish adds an OutboxMessage to the Actor's Outbox and sends notifications to all Followers.
func (service *Outbox) Publish(actor *outbox.Actor, parentType string, parentID primitive.ObjectID, activity mapof.Any) error {

	const location = "service.Outbox.Publish"

	// If we have anything BUT an "Update" activity, then write it to the Actor's Outbox
	if activity.GetString(vocab.PropertyType) == vocab.ActivityTypeCreate {

		if object, ok := activity[vocab.PropertyObject].(mapof.Any); ok {

			// Write a new OutboxMessage to the database
			outboxMessage := model.NewOutboxMessage()
			outboxMessage.ParentType = parentType
			outboxMessage.ParentID = parentID
			outboxMessage.URL = object.GetString(vocab.PropertyID)
			outboxMessage.ActivityType = object.GetString(vocab.PropertyType)

			if err := service.Save(&outboxMessage, "Publishing"); err != nil {
				return derp.Wrap(err, location, "Error saving outbox message", outboxMessage)
			}
			log.Trace().Str("id", outboxMessage.URL).Msg("Outbox Message saved")
		}
	}

	// Send notifications to all Followers
	go service.sendNotifications_ActivityPub(actor, activity)
	go service.sendNotifications_WebSub(parentType, parentID)
	go service.sendNotifications_WebMention(activity)
	go service.sendNotifications_Email(parentType, parentID, activity)

	// Success!!
	return nil
}

// UnPublish deletes an OutboxMessage from the Outbox, and sends notifications to all Followers
func (service *Outbox) UnPublish(actor *outbox.Actor, parentType string, parentID primitive.ObjectID, url string) error {

	// Load the Outbox Message
	message := model.NewOutboxMessage()
	if err := service.LoadByURL(parentType, parentID, url, &message); err != nil {
		if derp.IsNotFound(err) {
			log.Debug().Str("type", parentType).Str("parent", parentID.Hex()).Str("url", url).Msg("Outbox Message not found")
			return nil
		}
		return derp.Wrap(err, "service.Outbox.UnPublish", "Error loading outbox message", url)
	}

	// Delete the Message from the User's Outbox
	if err := service.Delete(&message, "Un-Publishing"); err != nil {
		return derp.Wrap(err, "service.Outbox.UnPublish", "Error deleting outbox message", message)
	}

	// Make a streams.Document from the URL
	document := service.activityService.NewDocument(mapof.Any{
		vocab.PropertyID: url,
	})

	// If the Message was a "Create" activity, then send a "Delete" activity to all followers
	if message.ActivityType == vocab.ActivityTypeCreate {
		log.Debug().Str("id", url).Msg("Sending Delete Activity")
		go actor.SendDelete(document)
		return nil
	}

	// Otherwise, send an "Undo" activity to all followers
	log.Debug().Str("id", url).Msg("Sending Undo Activity")
	go actor.SendUndo(document)
	return nil
}

/******************************************
 * Notification Protocols
 ******************************************/

// sendNotifications_ActivityPub sends ActivityPub updates to all Followers
func (service Outbox) sendNotifications_ActivityPub(actor *outbox.Actor, activity mapof.Any) {
	actor.Send(activity)
}

// TODO: HIGH: Thoroughly re-test WebSub notifications.  They've been rebuilt from scratch.
func (service Outbox) sendNotifications_WebSub(parentType string, parentID primitive.ObjectID) {

	const location = "service.Outbox.sendNotifications_WebSub"

	// Get this User's Followers from the database
	followers, err := service.followerService.WebSubFollowersChannel(parentType, parentID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading Followers", parentType, parentID))
	}

	// Queue up all ActivityPub messages to be sent
	for follower := range followers {

		task := queue.NewTask("SendWebSubMessage", mapof.Any{
			"inboxUrl": follower.Actor.InboxURL,
			"format":   follower.Format,
			"secret":   follower.Data.GetString("secret"),
		})

		if err := service.queue.Publish(task); err != nil {
			derp.Report(derp.Wrap(err, location, "Error publishing task", task))
		}
	}
}

// sendNotifications_WebMention sends WebMention updates to external websites that are
// mentioned in this stream.  This is here (and not in the outbox service)
// because we need to build the content in order to discover outbound links.
func (service *Outbox) sendNotifications_WebMention(activity mapof.Any) {

	const location = "service.Outbox.sendNotifications_WebMention"

	// Locate the object ID for this acticity
	object := activity.GetMap(vocab.PropertyObject)
	id := object.GetString(vocab.PropertyID)
	content := activity.GetString(vocab.PropertyContent)

	// Discover all webmention links in the content
	reader := strings.NewReader(content)
	links, err := webmention.DiscoverLinksFromReader(reader, id, "")

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error discovering webmention links", activity))
		return
	}

	// If no links, peace out, homie.
	if len(links) == 0 {
		return
	}

	// Add background tasks to TRY sending webmentions to every link we found
	for _, link := range links {

		task := queue.NewTask("SendWebMention", mapof.Any{
			"source": id,
			"target": link,
		})

		if err := service.queue.Publish(task); err != nil {
			derp.Report(derp.Wrap(err, location, "Error publishing task", task))
		}
	}
}

// sendNotifications_Email sends email notifications to all "email" Followers
func (service *Outbox) sendNotifications_Email(parentType string, parentID primitive.ObjectID, activity mapof.Any) {

	const location = "service.Outbox.sendNotifications_Email"

	followers, err := service.followerService.EmailFollowersChannel(parentType, parentID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading Followers", parentType, parentID))
		return
	}

	for follower := range followers {
		if err := service.domainEmail.SendFollowerActivity(follower, activity); err != nil {
			derp.Report(derp.Wrap(err, location, "Error sending email", follower))
		}
	}
}
