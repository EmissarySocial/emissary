package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/hannibal/streams"
	"github.com/labstack/echo/v4"
)

func GetFollowersCollection(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		result := streams.NewOrderedCollection()
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(http.StatusOK, result)
	}
}
