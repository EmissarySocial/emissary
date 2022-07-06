package middleware

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// Domain middleware validates that the requested domain has been fully activated before
// allowing the request to pass through.
func Domain(factory *server.Factory) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			domainFactory, err := factory.ByContext(ctx)

			if err != nil {
				return derp.NewForbiddenError("middleware.Domain", "Unrecognized domain", ctx.Request().URL.Hostname(), err)
			}

			if domainFactory.Session == nil {
				return derp.NewForbiddenError("middleware.Domain", "Database Not Configured for this Domain")
			}

			return next(ctx)
		}
	}
}
