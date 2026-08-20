package activitypub_stream

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// init registers the handlers that boost inbound activities to this Stream's followers
func init() {
	streamRouter.Add(vocab.ActivityTypeCreate, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeUndo, vocab.Any, BoostAny)

	streamRouter.Add(vocab.ActivityTypeAnnounce, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeLike, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeDislike, vocab.Any, BoostAny)
}

// BoostAny handles inbound Create/Update/Announce/Like/Dislike activities on a Stream: it caches
// the activity (or its object), projects a Like/Dislike/Announce into the Stream's response
// collection, and re-announces the activity to the Stream's followers.
func BoostAny(context Context, activity streams.Document) error {

	const location = "handler.activitypub_stream.BoostAny"

	// RULE: Require "boost-inbox" setting
	if !context.actor.BoostInbox {
		return derp.NotFound("activitypub_stream.inboxRouter", "Actor does not have an Inbox")
	}

	// RULE: If "followers-only" is set, then only accept activities from followers
	if context.actor.BoostFollowersOnly {
		if !context.factory.Follower().IsActivityPubFollower(context.session, model.FollowerTypeStream, context.stream.StreamID, activity.ActorID()) {
			return derp.Forbidden(location, "Must be a follower to post to this Actor", activity.ActorID())
		}
	}

	// RULE: never cache or amplify content from a blocked/muted actor. A Stream is a domain-level
	// actor, so this evaluates admin-tier rules (NilObjectID); announce() re-broadcasts to every
	// follower, making this the one amplification choke point (block AND mute both suppress it).
	disposition, err := context.factory.Rule().ActorDisposition(context.session, primitive.NilObjectID, activity, time.Now().Unix())

	if err != nil {
		return derp.Wrap(err, location, "Checking rules", activity.ActorID())
	}

	if disposition.IsFiltered() {
		return nil
	}

	activityService := context.factory.ActivityStream() // nolint:scopeguard (readability)

	switch activity.Type() {

	case vocab.ActivityTypeCreate:

		object := activity.Object()

		if err := activityService.Save(object); err != nil {
			return derp.Wrap(err, location, "Inserting object", object.ID())
		}
		return announce(context, object)

	case vocab.ActivityTypeUpdate:
		object := activity.Object()
		if err := activityService.Save(object); err != nil {
			return derp.Wrap(err, location, "Updating object", object.ID())
		}
		return nil

	case vocab.ActivityTypeAnnounce:
		object := activity.Object()
		if err := activityService.Save(object); err != nil {
			return derp.Wrap(err, location, "Saving object", object.ID())
		}
		return announce(context, object)

	default:
		if err := activityService.Save(activity); err != nil {
			return derp.Wrap(err, location, "Saving activity", activity.ID())
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
		return derp.Wrap(err, location, "Loading actor", context.stream)
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
		return derp.Wrap(err, location, "Saving message", context.stream.StreamID, activity.ID())
	}

	// Publish the Announce to the stream's followers as a post-commit send (F3, W6 option B). The
	// activity is addressed to the stream's followers collection, which SendLocator.Recipient
	// resolves; the signed delivery happens after commit and is per-recipient retryable.
	log.Debug().Msg("Announcing document to followers")
	announceID := context.stream.ActivityPubAnnouncedURL() + "/" + message.OutboxMessageID.Hex()
	followersURL := actor.ActorID() + "/pub/followers"
	outboxService.SendAnnounce(context.session, actor.ActorID(), announceID, activity, followersURL)

	return nil
}
