package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Blocked renders the page shown to a visitor whose request was refused by the Digital Dome
func Blocked(ctx echo.Context) error {
	return ctx.HTML(http.StatusNotFound, `<div style="height:100vh; display:flex; justify-content:center; align-items:center; font-size:128pt;"><span>🖕</span></div>`)
}
