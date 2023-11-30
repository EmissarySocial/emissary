package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeAnnounce, vocab.Any, receiveAnnounce)
}

// receiveAnnounce handles all Announce, Like, and Dislike activities
func receiveAnnounce(factory *domain.Factory, user *model.User, activity streams.Document) error {

	const location = "handler.receiveAnnounce"

	// RULE: Verify that the ActivityID exists
	if activity.ID() == "" {
		return derp.NewBadRequestError(location, activity.Type()+" activities must have an ID")
	}

	// Load the Activity from JSON-LD.
	// This populates the document in the ActivityStream cache
	activity, err := factory.ActivityStreams().Load(activity.ID())

	if err != nil {
		return derp.NewBadRequestError(location, "Unable to load Activity "+activity.ID())
	}

	// Last, preload the cache with the activity "Object", which is the document being Liked/Disliked
	_, _ = activity.Object().Load()

	// Success.
	return nil
}
