package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// GetInbox returns an inbox for a particular ACTOR
func GetInbox(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return nil
	}
}

// PostInbox accepts messages to a particular ACTOR
func PostInbox(fm *server.Factory) echo.HandlerFunc {

	const location = "whisperverse.handler.PostInbox"

	return func(ctx echo.Context) error {

		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized domain")
		}

		inboxService := factory.Inbox()

		body := make(map[string]interface{})

		if err := ctx.Bind(&body); err != nil {
			return derp.Wrap(err, location, "Error binding request body")
		}

		// TODO: Validate signatures here

		if err := inboxService.Receive(body); err != nil {
			return derp.Wrap(err, location, "Error processing ActivityPub message")
		}

		return ctx.NoContent(http.StatusNoContent)
	}
}
