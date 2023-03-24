package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/hannibal/streams"
	"github.com/labstack/echo/v4"
)

func ActivityPub_GetLikes(serverFactory *server.Factory) echo.HandlerFunc {

	// TODO: MEDIUM: This function should be implemented (once we have "likes")

	return func(ctx echo.Context) error {
		result := streams.NewOrderedCollection()
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(http.StatusOK, result)
	}
}
