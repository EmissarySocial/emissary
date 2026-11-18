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

	// RULE: A Create or Update carries only its actor's own objects, checked before any other work
	// so a forgery costs no rule lookups.  The activity's own id was bound in ReceiveRequest (D18).
	if err := requireOwnObject(activity); err != nil {
		return derp.Wrap(err, location, "Validating object origin", activity.ActorID())
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

		// RULE: An announced object usually lives on another origin, so the embedded copy is never
		// trusted.  Loading it by id fetches the copy its own host serves, and caches that one.
		object, err := activityService.AppClient().Load(activity.Object().ID())

		if err != nil {
			return derp.Wrap(err, location, "Loading announced object", activity.Object().ID())
		}

		return announce(context, object)

	default:
		if err := activityService.Save(activity); err != nil {
			return derp.Wrap(err, location, "Saving activity", activity.ID())
		}
		return announce(context, activity)
	}
}

// requireOwnObject returns a Forbidden error when a Create or Update carries an object whose id does
// not share its actor's origin, including an object with no id or a urn:uuid.
func requireOwnObject(activity streams.Document) error {

	const location = "handler.activitypub_stream.requireOwnObject"

	// Only Create and Update speak for their objects.  An Announce names someone else's, and is
	// loaded from its own host instead of trusted. (BUG-223)
	switch activity.Type() {
	case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate:
	default:
		return nil
	}

	if object := activity.Object(); !object.IsSameOrigin(activity.ActorID()) {
		return derp.Forbidden(location, "Object must share the actor's origin", activity.ActorID(), object.ID())
	}

	return nil
}

// announce boosts an activity to the Stream's followers, and keeps the Announce in the Stream's outbox
func announce(context Context, activity streams.Document) error {

	const location = "handler.activityPub_stream.announce"

	// Try to load the Actor for this Stream
	actor, err := context.ActivityPubActor()

	if err != nil {
		return derp.Wrap(err, location, "Loading actor", context.stream)
	}

	// Store the Announce in the Stream's outbox, and deliver it after this transaction commits
	log.Debug().Msg("Announcing document to followers")

	if _, err := context.factory.Outbox2().PublishAnnounce(context.session, model.ActorTypeStream, context.stream.StreamID, actor.ActorID(), activity); err != nil {
		return derp.Wrap(err, location, "Announcing activity", context.stream.StreamID)
	}

	return nil
}
