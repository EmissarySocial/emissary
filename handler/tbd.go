package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// TBD represents a placeholder handler that will be removed before release.
// It returns a 501 "Not Implemented" status code.
func TBD(ctx echo.Context) error {
	return ctx.String(http.StatusNotFound, "Not Found")
}
