package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Localhost is a middleware that guarantees that requests are coming from localhost.
func Localhost() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			if ctx.Request().Host != "localhost:8080" {
				return ctx.NoContent(http.StatusNotFound)
			}

			return next(ctx)
		}
	}
}
