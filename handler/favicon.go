package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// NotFound returns the default favicon for this server
func NotFound(ctx echo.Context) error {
	return derp.NewNotFoundError("", "")
}

// GetFavicon returns the default favicon for this server
func GetFavicon(factoryManager *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return derp.NewNotFoundError("", "")
	}
}
