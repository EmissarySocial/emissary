package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/labstack/echo/v4"
)

func ActivityPub_GetBlocks(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetBlocks"

	return func(ctx echo.Context) error {
		// TODO: CRITICAL: Implement this
		return nil
	}
}
