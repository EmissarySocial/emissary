package handler

import (
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

// TBD represents a placeholder handler that will be removed before release.
func TBD(ctx echo.Context) error {
	return ctx.String(http.StatusOK, spew.Sdump(ctx.Request()))
}
