package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
	"github.com/labstack/echo/v4"
)

func PostInbox(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.activitypub.ActivityPub_PostInbox"

	return func(ctx echo.Context) error {

		// Retrieve the Factory and Stream from the Request
		factory, _, _, _, stream, actor, err := getActor(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Request")
		}

		// Retrieve the activity from the request body
		activity, err := pub.ReceiveInboxRequest(ctx.Request(), factory.ActivityStreams())

		if err != nil {
			return derp.Wrap(err, location, "Error parsing ActivityPub request")
		}

		// Create a new request context for the ActivityPub router
		context := Context{
			factory: factory,
			stream:  &stream,
			actor:   &actor,
		}

		// Handle the ActivityPub request
		if err := streamRouter.Handle(context, activity); err != nil {
			return derp.Wrap(err, location, "Error handling ActivityPub request")
		}

		// Send the response to the client
		return ctx.String(http.StatusOK, "")
	}
}
