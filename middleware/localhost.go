package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Localhost is a middleware that guarantees that requests are coming specifically "localhost".
// Not just any local domain, and not using a proxied "x-Forwarded-Host" value, but specifically, exaclty "localhost".
func Localhost() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			hostname := ctx.Request().Host

			if hostname != "localhost" {
				return ctx.NoContent(http.StatusNotFound)
			}

			return next(ctx)
		}
	}
}
