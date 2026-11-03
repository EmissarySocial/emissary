package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// APINotFound answers an API path that no handler implements with a Mastodon-style JSON 404.
func APINotFound(ctx echo.Context) error {

	// These aren't the droids you're looking for
	return ctx.JSON(http.StatusNotFound, map[string]string{"error": "Not found"})
}
