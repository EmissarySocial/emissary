package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/jsonld"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/vocab"
	"github.com/labstack/echo/v4"
)

func ActivityPub_PostInbox(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_PostInbox"

	return func(ctx echo.Context) error {

		// Find the factory for this hostname
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, location, "Error creating ActivityStreamsHandler"))
		}

		// Retrieve the activity from the request body
		activityType, objectType, activity, err := pub.ParseInboxRequest(ctx.Request(), factory.JSONLDClient())

		if err != nil {
			return derp.Report(derp.Wrap(err, location, "Error parsing ActivityPub request"))
		}

		// Handle the ActivityPub request
		if err := activityPub_inbox(factory, activityType, objectType, activity); err != nil {
			return derp.Report(derp.Wrap(err, location, "Error handling ActivityPub request"))
		}

		// Send the response to the client
		return ctx.String(http.StatusOK, "")
	}
}

func activityPub_inbox(factory *domain.Factory, activityType string, objectType string, activity jsonld.Reader) error {

	const location = "handler.activityPub_inbox"

	switch activityType {

	case vocab.ActivityTypeAccept:
		return activityPub_inbox_Accept(factory, activity)

	case vocab.ActivityTypeFollow:
		return activityPub_inbox_Follow(factory, activity)

	}

	return derp.NewBadRequestError(location, "Activity type not supported", activityType, activity.Value())
}
