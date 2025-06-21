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
	inboxRouter.Add(vocab.ActivityTypeDislike, vocab.Any, receiveLikeOrAnnounce)
}

// receiveLikeOrAnnounce handles all Like Dislike activities
func receiveLikeOrAnnounce(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.receiveLikeOrAnnounce"

	// Load the original object being Announced/Liked
	object, err := activity.Object().Load()

	if err != nil {
		return derp.Wrap(err, location, "Error loading object", activity.Object().ID())
	}

	// Add then Shared/Liked Object into the ActivityStream cache
	if err := inboxRouter.Handle(context, object); err != nil {
		return derp.Wrap(err, location, "Error processing activity Object", activity.Object().ID())
	}

	// RULE: If the Activity does not have an ID, then make a new "fake" one.
	if activity.ID() == "" {
		activity.SetProperty(vocab.PropertyID, activitypub.FakeActivityID(activity))
	}

	// Add the Announce/Like/Dislike into the ActivityStream cache
	context.factory.ActivityStream().Put(activity)

	// Add the activity into the User's Inbox
	if err := saveMessage(context, object, activity.Actor().ID(), getOriginType(activity.Type())); err != nil {
		return derp.Wrap(err, location, "Error saving message", context.user.UserID, activity.Value())
	}

	// Success.
	return nil
}

// saveMessage saves a message into the User's inbox
func saveMessage(context Context, activity streams.Document, actorID string, originType string) error {

	const location = "handler.activitypub_user.saveMessage"

	// Verify that this message comes from a valid "Following" object.
	followingService := context.factory.Following()
	following := model.NewFollowing()

	// If the "Following" record cannot be found, then do not add a message
	if err := followingService.LoadByURL(context.user.UserID, actorID, &following); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error loading Following record", context.user.UserID, actorID)
	}

	// Try to save the message to the database (with de-duplication)
	if err := followingService.SaveMessage(&following, activity, originType); err != nil {
		return derp.Wrap(err, location, "Error saving message", context.user.UserID, activity.Value())
	}

	// Success.
	return nil

}
