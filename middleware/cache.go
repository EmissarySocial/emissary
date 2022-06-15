package middleware

import "github.com/labstack/echo/v4"

func CacheControl(cacheControl string) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {
			ctx.Response().Header().Set("Cache-Control", cacheControl)
			return next(ctx)
		}
	}
}
