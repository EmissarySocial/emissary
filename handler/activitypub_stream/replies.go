package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/labstack/echo/v4"
)

func GetRepliesCollection(serverFactory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return nil
	}
}
