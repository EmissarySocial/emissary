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
		if !context.factory.Follower().IsActivityPubFollower(context.session, model.FollowerTypeStream, context.stream.StreamID, activity.Actor().ID()) {
			return derp.ForbiddenError(location, "Must be a follower to post to this Actor", activity.Actor().ID())
		}
	}

	// Get an ActivityStream service for the Stream
	activityService := context.factory.ActivityStream(model.ActorTypeStream, context.stream.StreamID)

	switch activity.Type() {

	case vocab.ActivityTypeCreate:
		object := activity.Object()
		if err := activityService.Save(object); err != nil {
			return derp.Wrap(err, location, "Unable to insert object", object.ID())
		}
		return announce(context, object)

	case vocab.ActivityTypeUpdate:
		object := activity.Object()
		if err := activityService.Save(object); err != nil {
			return derp.Wrap(err, location, "Unable to update object", object.ID())
		}
		return nil

	case vocab.ActivityTypeAnnounce:
		object := activity.Object()
		if err := activityService.Save(object); err != nil {
			return derp.Wrap(err, location, "Unable to save object", object.ID())
		}
		return announce(context, object)

	default:
		if err := activityService.Save(activity); err != nil {
			return derp.Wrap(err, location, "Unable to save activity", activity.ID())
		}
		return announce(context, activity)
	}
}

// announce saves the activity into the Stream's outbox
func announce(context Context, activity streams.Document) error {

	const location = "handler.activityPub_stream.announce"

	// Try to load the Actor for this Stream
	actor, err := context.ActivityPubActor()

	if err != nil {
		return derp.Wrap(err, location, "Unable to load actor", context.stream)
	}

	// Convert the Activity into an Inbox Message
	message := model.NewOutboxMessage()
	message.ActorID = context.stream.StreamID
	message.ActorType = model.FollowerTypeStream
	message.ActivityType = activity.Type()
	message.ObjectID = activity.ID()

	// Try to save the message to the content Actor's outbox
	outboxService := context.factory.Outbox()
	if err := outboxService.Save(context.session, &message, "via ActivityPub"); err != nil {
		return derp.Wrap(err, location, "Unable to save message", context.stream.StreamID, activity.ID())
	}

	// Send the Announce to all of our followers
	log.Debug().Msg("Announcing document to followers")
	announceID := context.stream.ActivityPubAnnouncedURL() + "/" + message.OutboxMessageID.Hex()
	actor.SendAnnounce(announceID, activity)

	return nil
}
