package middleware

import "github.com/labstack/echo/v4"

// MimeType generates an echo.Middleware function that forces the "Accept:" header
// to match a particular mime type.  This is useful for switching the KIND of data
// returned by the server, based on URL, or Accept headers.
func MimeType(mimeType string) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			ctx.Request().Header.Set("Accept", mimeType)
			return next(ctx)
		}
	}
}
