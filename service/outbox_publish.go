package service

import (
	"iter"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/streams"
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
func (service *Outbox) Publish(actor *outbox.Actor, actorType string, actorID primitive.ObjectID, activity streams.Document, permissions model.Permissions) error {

	const location = "service.Outbox.Publish"

	// Write a new OutboxMessage to the database
	outboxMessage := model.NewOutboxMessage()
	outboxMessage.ActorType = actorType
	outboxMessage.ActorID = actorID
	outboxMessage.ObjectID = activity.Object().ID()
	outboxMessage.ActivityType = activity.Type()
	outboxMessage.Permissions = permissions

	if err := service.Save(&outboxMessage, "Publishing"); err != nil {
		return derp.Wrap(err, location, "Error saving outbox message", outboxMessage)
	}

	log.Trace().Str("id", outboxMessage.ObjectID).Msg("Outbox Message saved.  Notifying Followers")

	// Get All Followers for this Actor and Addressees
	recipients := joinIterators(
		service.followerService.RangeFollowers(actorType, actorID),
		service.addresseesAsFollowers(activity.RangeAddressees()),
		service.addresseesAsFollowers(activity.RangeInReplyTo()),
		// TODO: service.webMentionsAsFollowers(activity),
	)

	ruleFilter := service.ruleService.Filter(actorID, WithBlocksOnly())
	activityMap := activity.Map()

	for follower := range recipients {

		// Do not send to blocked Followers
		if !ruleFilter.AllowSend(follower.Actor.ProfileURL) {
			continue
		}

		// Do not send to Followers who do not have permissions to view this activity
		if !service.identityService.HasPermissions(follower.Method, follower.Actor.ProfileURL, permissions) {
			continue
		}

		switch follower.Method {

		case model.FollowerMethodActivityPub:
			service.sendNotification_ActivityPub(actor, &follower, activityMap)

		case model.FollowerMethodWebSub:
			service.sendNotification_WebSub(&follower)

		case model.FollowerMethodEmail:
			service.sendNotification_Email(&follower, activityMap)

		// TODO: Can we move WebMentions into this too?
		default:
			derp.Report(derp.InternalError(location, "Unknown Follower Method.  This should never happen", follower))
		}
	}

	// Send notifications to all Followers
	go service.sendNotifications_WebMention(activityMap)

	// Success!!
	return nil
}

// UnPublish deletes an OutboxMessage from the Outbox, and sends notifications to all Followers
func (service *Outbox) UnPublish(actor *outbox.Actor, actorType string, actorID primitive.ObjectID, url string) error {

	// Load the Outbox Message
	message := model.NewOutboxMessage()
	if err := service.LoadByURL(actorType, actorID, url, &message); err != nil {
		if derp.IsNotFound(err) {
			log.Debug().Str("type", actorType).Str("parent", actorID.Hex()).Str("url", url).Msg("Outbox Message not found")
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

func (service *Outbox) addresseesAsFollowers(addressees iter.Seq[string]) iter.Seq[model.Follower] {

	return func(yield func(model.Follower) bool) {

		uniquer := streams.NewUniquer[string]()

		for addressee := range uniquer.Range(addressees) {
			follower := model.NewFollower()
			follower.Actor.ProfileURL = addressee
			follower.Method = model.FollowerMethodActivityPub
			follower.StateID = model.FollowerStateActive

			if !yield(follower) {
				return
			}
		}
	}
}

// sendNotifications_ActivityPub sends ActivityPub updates to all Followers
// TODO: HIGH: This should be a background task with retries, just like sendNotification_WebSub
func (service Outbox) sendNotification_ActivityPub(actor *outbox.Actor, follower *model.Follower, activity mapof.Any) {
	if err := actor.SendOne(follower.Actor.ProfileURL, activity); err != nil {
		derp.Report(derp.Wrap(err, "service.Outbox.sendNotifications_ActivityPub", "Error sending ActivityPub notification", follower.Actor.ProfileURL))
	}
}

// TODO: HIGH: Thoroughly re-test WebSub notifications.  They've been rebuilt from scratch.
func (service Outbox) sendNotification_WebSub(follower *model.Follower) {

	const location = "service.Outbox.sendNotifications_WebSub"

	task := queue.NewTask("SendWebSubMessage", mapof.Any{
		"inboxUrl": follower.Actor.InboxURL,
		"format":   follower.Format,
		"secret":   follower.Data.GetString("secret"),
	})

	if err := service.queue.Publish(task); err != nil {
		derp.Report(derp.Wrap(err, location, "Error publishing task", task))
	}
}

// sendNotifications_Email sends email notifications to all "email" Followers
func (service *Outbox) sendNotification_Email(follower *model.Follower, activity mapof.Any) {

	const location = "service.Outbox.sendNotifications_Email"

	if err := service.domainEmail.SendFollowerActivity(follower, activity); err != nil {
		derp.Report(derp.Wrap(err, location, "Error sending email", follower))
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
