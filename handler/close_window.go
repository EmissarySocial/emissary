package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetCloseWindow is an HTTP handler that returns HTML to close the current window.
// It is used as a callback URL for outbound intents that use true pop-up windows.
func GetCloseWindow(ctx echo.Context) error {
	return ctx.HTML(http.StatusOK, `<html><head><script>window.close();</script></head></html>`)
}
