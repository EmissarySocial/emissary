package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
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
		activity, err := pub.ParseInboxRequest(ctx.Request(), factory.HTTPCache())

		if err != nil {
			return derp.Report(derp.Wrap(err, location, "Error parsing ActivityPub request"))
		}

		// Handle the ActivityPub request
		if err := inboxRouter.Handle(factory, activity); err != nil {
			return derp.Report(derp.Wrap(err, location, "Error handling ActivityPub request"))
		}

		// Send the response to the client
		return ctx.String(http.StatusOK, "")
	}
}
