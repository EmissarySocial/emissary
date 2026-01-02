package activitypub_user

import (
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeAnnounce, vocab.Any, receiveLikeOrAnnounce)
	inboxRouter.Add(vocab.ActivityTypeLike, vocab.Any, receiveLikeOrAnnounce)

	// Removing Dislikes for now... Semantics on this are unclear, but Dislikes
	// probably SHOULD NOT end up in a user's inbox.
	// inboxRouter.Add(vocab.ActivityTypeDislike, vocab.Any, receiveLikeOrAnnounce)
}

// receiveLikeOrAnnounce handles all Like Dislike activities
func receiveLikeOrAnnounce(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.receiveLikeOrAnnounce"

	// RULE: If the Activity does not have an ID, then make a new "fake" one.
	if activity.ID() == "" {
		activity.SetProperty(vocab.PropertyID, activitypub.FakeActivityID(activity))
	}

	// Collect the ActorID for this Activity
	actorID := activity.Actor().ID()

	if actorID == "" {
		return derp.BadRequest(location, "Activity must have an ActorID", activity.Value())
	}

	// Verify that this message comes from an actor that we're "Following"
	followingService := context.factory.Following()
	following := model.NewFollowing()

	if err := followingService.LoadByURL(context.session, context.user.UserID, actorID, &following); err != nil {
		return derp.Wrap(err, location, "Unable to locate Following record", activity.Value())
	}

	// Load the original ActivityStream document being Announced/Liked (which also adds it to the cache)
	document, err := activity.Object().Load()

	if err != nil {
		return derp.Wrap(err, location, "Unable to load ActivityStream document", activity.Object().ID())
	}

	// Add the activity into the ActivityStream cache (for statistics)
	if err := context.factory.ActivityStream().Save(activity); err != nil {
		return derp.Wrap(err, location, "Unable to save activity", activity.ID())
	}

	originType := getOriginType(activity.Type())

	// Add the Announced/Liked message into the User's inbox
	if err := followingService.SaveMessage(context.session, &following, document, originType); err != nil {
		return derp.Wrap(err, location, "Unable to save message", context.user.UserID, activity.Value())
	}

	// Success.
	return nil
}
