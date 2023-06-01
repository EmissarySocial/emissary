package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

func ActivityPub_GetOutbox(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetOutbox"

	return func(ctx echo.Context) error {
		// TODO: CRITICAL: Implement ActivityPub outbox
		return derp.NewBadRequestError(location, "Not implemented")
	}
}
