package middleware

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// Domain middleware validates that the requested domain has been fully activated before
// allowing the request to pass through.
func Domain(factory *server.Factory) echo.MiddlewareFunc {

	const location = "middleware.Domain"

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			// Locate the Domain factory for this request
			domainFactory, err := factory.ByContext(ctx)

			if err != nil {
				return derp.NewMisdirectedRequestError(location, "Invalid hostname", ctx.Path(), err)
			}

			// Guarantee that the database session is not nil
			if domainFactory.Session == nil {
				return derp.NewInternalError(location, "Database not ready")
			}

			return next(ctx)
		}
	}
}
