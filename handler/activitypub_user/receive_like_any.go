package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeAnnounce, vocab.Any, receiveResponse)
	inboxRouter.Add(vocab.ActivityTypeLike, vocab.Any, receiveResponse)
	inboxRouter.Add(vocab.ActivityTypeDislike, vocab.Any, receiveResponse)
}

// receiveResponse handles all Announce, Like, and Dislike activities
func receiveResponse(context Context, activity streams.Document) error {

	const location = "handler.activitypub.receiveResponse"

	// RULE: Verify that the ActivityID exists
	if activity.ID() == "" {
		return derp.NewBadRequestError(location, activity.Type()+" activities must have an ID")
	}

	// Add the Like/Dislike into the ActivityStream cache
	context.factory.ActivityStreams().Put(activity)

	// Add the Liked/Disliked document into the ActivityStream cache
	document := activity.UnwrapActivity()
	document, err := document.Load()

	if err != nil {
		return derp.Wrap(err, location, "Error loading document from ActivityStreams", document.Value())
	}

	// Verify that this message comes from a valid "Following" object.
	followingService := context.factory.Following()
	following := model.NewFollowing()

	// If the "Following" record cannot be found, then do not add a message
	if err := followingService.LoadByURL(context.user.UserID, activity.Actor().ID(), &following); err != nil {
		return nil
	}

	// Calculate the origin type (ANNOUNCE, LIKE, or DISLIKE)
	originType := getOriginType(activity.Type())

	// Try to save the message to the database (with de-duplication)
	if err := followingService.SaveMessage(&following, document, originType); err != nil {
		return derp.Wrap(err, "handler.activitypub_receive_create", "Error saving message", context.user.UserID, document.Object().ID())
	}

	// Success.
	return nil
}

// getOriginType translates from ActivityStreams.Type => model.OriginType constants
func getOriginType(activityType string) string {

	switch activityType {

	case vocab.ActivityTypeAnnounce:
		return model.OriginTypeAnnounce

	case vocab.ActivityTypeLike:
		return model.OriginTypeLike

	case vocab.ActivityTypeDislike:
		return model.OriginTypeDislike
	}

	return model.OriginTypePrimary
}
