package handler

import (
	"github.com/EmissarySocial/emissary/domain"
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
func receiveResponse(factory *domain.Factory, user *model.User, activity streams.Document) error {

	const location = "handler.receiveResponse"

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

	// Success.
	return nil
}
