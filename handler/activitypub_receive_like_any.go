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

	objectURL := activity.Object().String()

	// Load Actor information from JSON-LD
	actor, err := activity.Actor().Load()

	if err != nil {
		return derp.Wrap(err, "handler.receiveResponse", "Error loading actor")
	}

	// Create an Origin link for this activity
	responseType, responseLabel := parseResponse(activity)
	origin := model.OriginLink{
		Type:    model.OriginTypeActivityPub,
		URL:     activity.ID(),
		Label:   responseType,
		Summary: responseLabel + " by " + actor.Name(),
	}

	// If the activity's Object is a local stream, then add a StreamResponse to it...
	streamService := factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByURL(objectURL, &stream); err == nil {

		streamResponseService := factory.StreamResponse()

		// Create/Update a StreamResponse for this activity
		if err := streamResponseService.SetStreamResponse(&stream, origin, parseActor(actor), responseType, ""); err != nil {
			return derp.Wrap(err, "handler.receiveResponse", "Error saving StreamResponse")
		}

		return nil
	}

	// If we have already received the activity's Object in the Inbox, then add a Response to it...
	inboxService := factory.Inbox()
	message := model.NewMessage()

	if inboxService.LoadByURL(user.UserID, objectURL, &message) == nil {

		responseService := factory.Response()

		if err := responseService.SetResponse(&message, parseActor(actor), responseType, ""); err != nil {
			return derp.Wrap(err, "handler.receiveResponse", "Error saving Response")
		}

		return nil
	}

	// Otherwise, this is for an external ActivityPub document -- add it to the Inbox (as a "share")
	/*
		message := model.NewMessage()
		message.Origin = "Liked by ...", "Mentioned by ..."
	*/

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

// parseActor generates a PersonLink using an ActivityPub `Actor` document
func parseActor(actor streams.Document) model.PersonLink {

	return model.PersonLink{
		Name:       actor.Name(),
		ProfileURL: actor.ID(),
		ImageURL:   actor.ImageURL(),
	}
}
