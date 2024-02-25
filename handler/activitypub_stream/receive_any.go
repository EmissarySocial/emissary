package activitypub_stream

import (
	"fmt"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

func init() {
	fmt.Println("Initializing BoostAny")
	streamRouter.Add(vocab.ActivityTypeCreate, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeUndo, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeDelete, vocab.Any, BoostAny)

	streamRouter.Add(vocab.ActivityTypeAnnounce, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeLike, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeDislike, vocab.Any, BoostAny)
}

func BoostAny(context Context, activity streams.Document) error {

	const location = "activitypub_stream.inboxRouter.BoostAny"

	log.Debug().Str("activity", activity.ID()).Msg("Stream Inbox: Received new Activity")

	// RULE: Require  "boost-inbox" setting
	if !context.actor.BoostInbox {
		return derp.NewNotFoundError("activitypub_stream.inboxRouter", "Actor does not have an Inbox")
	}

	// RULE: If "followers-only" is set, then only accept activities from followers
	if context.actor.BoostFollowersOnly {
		if !context.factory.Follower().IsActivityPubFollower(context.stream.StreamID, activity.Actor().ID()) {
			return derp.NewForbiddenError(location, "Must be a follower to post to this Actor", activity.Actor().ID())
		}
	}

	// Save the activity into the cache
	activityService := context.factory.ActivityStream()

	switch activity.Type() {

	case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate, vocab.ActivityTypeDelete:
		object := activity.Object()
		activityService.Put(object)

	case vocab.ActivityTypeAnnounce:
		object := activity.Object()
		activityService.Put(object)
		activityService.Put(activity)

	default:
		activityService.Put(activity)
	}

	// Try to load the Actor for this user
	activityPubActor, err := context.factory.Stream().ActivityPubActor(context.stream.StreamID, true)

	if err != nil {
		return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading actor", context.stream)
	}

	announceID := context.stream.ActivityPubAnnouncedURL() + "/" + activitypub.FakeActivityID(activity)

	// Send the Announce to all of our followers
	log.Debug().Msg("Announcing document to followers")
	activityPubActor.SendAnnounce(announceID, activity)
	return nil
}
