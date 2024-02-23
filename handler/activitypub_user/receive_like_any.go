package activitypub_user

import (
	"crypto/sha256"

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

	// Add then Shared/Liked Object into the ActivityStream cache
	if err := inboxRouter.Handle(context, activity.Object()); err != nil {
		return derp.Wrap(err, location, "Error processing activity Object", activity.Object().ID())
	}

	// RULE: If the Activity does not have an ID, then make a new "fake" one.
	if activity.ID() == "" {
		activity.SetProperty(vocab.PropertyID, fakeResponseID(activity))
	}

	// Add the Announce/Like/Dislike into the ActivityStream cache
	context.factory.ActivityStreams().Put(activity)

	// Add the activity into the User's Inbox
	if err := saveMessage(context, activity, activity.Actor().ID(), getOriginType(activity.Type())); err != nil {
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

		if derp.NotFound(err) {
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

// fakeResponseID generates a unique ID for a stream document based on
// the hashed contents of the document.
func fakeResponseID(activity streams.Document) string {
	plainText := activity.Type() + " " + activity.Object().ID() + " " + activity.Actor().ID()
	hasher := sha256.New()
	hasher.Write([]byte(plainText))
	return "sha256-" + string(hasher.Sum(nil))
}
