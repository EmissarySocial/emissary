package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeLike, vocab.Any, activityPub_LikeOrDislike)
}

func activityPub_LikeOrDislike(factory *domain.Factory, user *model.User, activity streams.Document) error {

	// Get required services
	responseService := factory.Response()
	locatorService := factory.Locator()

	// Try to find the object that is being responded to
	object, err := locatorService.GetDocumentFromURL(activity.Object())

	if err != nil {
		return derp.Wrap(err, "handler.activitypub_receive_create", "Error loading object", user.UserID, activity.Object().ID())
	}

	// Create the response object
	response := model.NewResponse()
	response.Actor = convert.ActivityPubPersonLink(activity.Actor())
	response.Object = object

	switch activity.Type() {

	case vocab.ActivityTypeLike:
		response.Type = model.ResponseTypeLike

	case vocab.ActivityTypeDislike:
		response.Type = model.ResponseTypeDislike

	default:
		return derp.NewInternalError("handler.activitypub_receive_create", "Invalid activity type", activity.Type())
	}

	// Save the response object
	if err := responseService.Save(&response, "Created via ActivityPub"); err != nil {
		return derp.Wrap(err, "handler.activitypub_receive_create", "Error saving response", user.UserID, activity.Object().ID())
	}

	return nil
}
