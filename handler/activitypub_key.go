package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/labstack/echo/v4"
)

func ActivityPub_GetPublicKey(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetPublicKey"

	return func(ctx echo.Context) error {
		// TODO: CRITICAL: Implement this
		return nil
	}
}
