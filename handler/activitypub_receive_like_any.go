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

func receiveResponse(factory *domain.Factory, user *model.User, activity streams.Document) error {

	// Load Actor information from JSON-LD
	actor, err := activity.Actor().Load()

	if err != nil {
		return derp.Wrap(err, "handler.receiveResponse", "Error loading actor")
	}

	// Parse respone type
	responseType, responseLabel := parseResponse(activity)

	// Create the response
	response := model.NewResponse()
	response.URL = activity.ID()
	response.ActorID = actor.ID()
	response.Type = responseType
	response.ObjectID = activity.Object().ID()
	response.Content = activity.Content()
	response.Summary = responseLabel + " by " + actor.Name()

	// Save/Update the response
	responseService := factory.Response()
	if err := responseService.SetResponse(&response); err != nil {
		return derp.Wrap(err, "handler.receiveResponse", "Error saving Response")
	}

	return nil
}

// parseReponse returns the response type and value for a given ActivityPub document
func parseResponse(activity streams.Document) (string, string) {

	switch activity.Type() {

	case vocab.ActivityTypeLike:
		return model.ResponseTypeLike, "Liked"

	case vocab.ActivityTypeDislike:
		return model.ResponseTypeDislike, "Disliked"

	case vocab.ActivityTypeAnnounce:
		return model.ResponseTypeShare, "Shared"
	}

	return model.ResponseTypeLike, "Mentioned"
}
