package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetEmptyList answers a Mastodon list endpoint that toot has no route for with an empty list
func GetEmptyList(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, []struct{}{})
}

// GetStreamingHealth answers Mastodon's streaming health check
func GetStreamingHealth(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "OK")
}
