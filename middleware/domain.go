package middleware

import (
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// Domain middleware validates that the requested domain has been fully activated before
// allowing the request to pass through.
func Domain(factory *server.Factory) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			domainFactory, err := factory.ByContext(ctx)

			if err != nil {
				return derp.NewForbiddenError("whisperverse.middleware.Domain", "Unrecognized domain", ctx.Request().URL.Hostname(), err)
			}

			if domainFactory.Session == nil {
				return derp.NewForbiddenError("whisperverse.middleware.Domain", "Database Not Configured for this Domain")
			}

			return next(ctx)
		}
	}
}
