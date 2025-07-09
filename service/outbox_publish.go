package service

import (
	"iter"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/davecgh/go-spew/spew"
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

	log.Trace().Str("objectId", outboxMessage.ObjectID).Msg("Outbox Message saved.  Notifying Followers")

	// Get All Followers for this Actor and Addressees
	recipients := joinIterators(
		service.followerService.RangeFollowers(actorType, actorID),
		service.addresseesAsFollowers(activity.RangeAddressees()),
		service.addresseesAsFollowers(activity.RangeInReplyTo()),
		// TODO: service.webMentionsAsFollowers(activity),
	)

	ruleFilter := service.ruleService.Filter(actorID, WithBlocksOnly())
	activityMap := activity.Map()
	activityMap[vocab.PropertyID] = outboxMessage.ActivityPubURL()

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

func (service *Outbox) DeleteActivity(actor *outbox.Actor, actorType string, actorID primitive.ObjectID, objectID string, permissions model.Permissions) error {
	return service.unpublish(actor, actorType, actorID, vocab.ActivityTypeDelete, objectID, permissions)
}

func (service *Outbox) UndoActivity(actor *outbox.Actor, actorType string, actorID primitive.ObjectID, objectID string, permissions model.Permissions) error {
	return service.unpublish(actor, actorType, actorID, vocab.ActivityTypeUndo, objectID, permissions)
}

// UnPublish deletes an OutboxMessage from the Outbox, and sends notifications to all Followers
func (service *Outbox) unpublish(actor *outbox.Actor, actorType string, actorID primitive.ObjectID, activityType string, objectID string, permissions model.Permissions) error {

	const location = "service.Outbox.unpublish"

	spew.Dump(location, actorType, actorID.Hex(), activityType, objectID)

	// Find all activities in the outbox related to this activity
	activities, err := service.RangeByObjectID(actorType, actorID, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load outbox activity", objectID)
	}

	// Remove each outbox activity
	for activity := range activities {

		// Delete the Activity from the User's Outbox
		if err := service.Delete(&activity, "Un-Publishing"); err != nil {
			return derp.Wrap(err, location, "Unable to delete outbox activity", activity)
		}
	}

	// TODO: This should also support "Undo" activities in the future,
	// but this will require additional function arguments.

	// Make a streams.Document to represent the "Delete" activity
	document := service.activityService.NewDocument(mapof.Any{
		vocab.PropertyActor:     actor.ActorID(),
		vocab.PropertyType:      activityType,
		vocab.PropertyObject:    objectID,
		vocab.PropertyPublished: hannibal.TimeFormat(time.Now()),
	})

	// Publish the "Delete" activity to the Outbox
	if err := service.Publish(actor, actorType, actorID, document, permissions); err != nil {
		return derp.Wrap(err, location, "Unable to publish DELETE activity", objectID)
	}

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
