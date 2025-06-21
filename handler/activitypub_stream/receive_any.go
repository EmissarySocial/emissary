package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

func init() {
	streamRouter.Add(vocab.ActivityTypeCreate, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeUndo, vocab.Any, BoostAny)

	streamRouter.Add(vocab.ActivityTypeAnnounce, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeLike, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeDislike, vocab.Any, BoostAny)
}

func BoostAny(context Context, activity streams.Document) error {

	const location = "handler.activitypub_stream.BoostAny"

	// RULE: Require "boost-inbox" setting
	if !context.actor.BoostInbox {
		return derp.NotFoundError("activitypub_stream.inboxRouter", "Actor does not have an Inbox")
	}

	// RULE: If "followers-only" is set, then only accept activities from followers
	if context.actor.BoostFollowersOnly {
		if !context.factory.Follower().IsActivityPubFollower(model.FollowerTypeStream, context.stream.StreamID, activity.Actor().ID()) {
			return derp.ForbiddenError(location, "Must be a follower to post to this Actor", activity.Actor().ID())
		}
	}

	// Rules for different activity types
	activityService := context.factory.ActivityStream()

	switch activity.Type() {

	case vocab.ActivityTypeCreate:
		object := activity.Object()
		activityService.Put(object)
		return announce(context, object)

	case vocab.ActivityTypeUpdate:
		object := activity.Object()
		activityService.Put(object)
		return nil

	case vocab.ActivityTypeAnnounce:
		activityService.Put(activity)
		object := activity.Object()
		activityService.Put(object)
		return announce(context, object)

	default:
		activityService.Put(activity)
		return announce(context, activity)
	}
}

// announce saves the activity into the Stream's outbox
func announce(context Context, activity streams.Document) error {

	// Try to load the Actor for this Stream
	actor, err := context.ActivityPubActor(true)

	if err != nil {
		return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading actor", context.stream)
	}

	// Convert the Activity into an Inbox Message
	message := model.NewOutboxMessage()
	message.ParentID = context.stream.StreamID
	message.ParentType = model.FollowerTypeStream
	message.ActivityType = activity.Type()
	message.URL = activity.ID()

	// Try to save the message to the content Actor's outbox
	outboxService := context.factory.Outbox()
	if err := outboxService.Save(&message, "via ActivityPub"); err != nil {
		return derp.Wrap(err, "activitypub_stream.saveMessage", "Error saving message", context.stream.StreamID, activity.ID())
	}

	// Send the Announce to all of our followers
	log.Debug().Msg("Announcing document to followers")
	announceID := context.stream.ActivityPubAnnouncedURL() + "/" + message.OutboxMessageID.Hex()
	actor.SendAnnounce(announceID, activity)

	return nil
}
