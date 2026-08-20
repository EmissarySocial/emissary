package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// RedirectTo returns a handler that sends every request to a fixed location
func RedirectTo(location string) func(ctx echo.Context) error {

	return func(ctx echo.Context) error {
		return ctx.Redirect(http.StatusSeeOther, location)
	}
}
