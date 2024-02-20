package activitypub_stream

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	streamRouter.Add(vocab.ActivityTypeCreate, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeLike, vocab.Any, BoostAny)
	streamRouter.Add(vocab.ActivityTypeDislike, vocab.Any, BoostAny)
}

func BoostAny(context Context, activity streams.Document) error {

	const location = "activitypub_stream.inboxRouter.BoostAny"

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
	object := activity.UnwrapActivity()
	context.factory.ActivityStreams().Put(object)

	// Try to load the Actor for this user
	activityPubActor, err := context.factory.Stream().ActivityPubActor(context.stream, true)

	if err != nil {
		return derp.Wrap(err, "handler.activityPub_HandleRequest_Follow", "Error loading actor", context.stream)
	}

	// Send the Announce to all of our followers
	activityPubActor.SendAnnounce(activity)
	return nil
}
