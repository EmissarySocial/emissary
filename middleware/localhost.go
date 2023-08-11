package middleware

import (
	"net/http"

	"github.com/benpate/rosetta/list"
	"github.com/labstack/echo/v4"
)

// Localhost is a middleware that guarantees that requests are coming from localhost.
func Localhost() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			hostname := list.First(ctx.Request().Host, ':')

			if hostname != "localhost" {
				return ctx.NoContent(http.StatusNotFound)
			}

			return next(ctx)
		}
	}
}
