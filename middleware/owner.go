package middleware

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// Owner middleware guarantees that the request is being performed by a website owner
func Owner(next echo.HandlerFunc) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Guarantee that we have a steranko.Context
		if sterankoContext, ok := ctx.(*steranko.Context); ok {

			// Try to get the authorization from the context
			authorization, err := sterankoContext.Authorization()

			if err != nil {
				// If not authorized, return NOT AUTHORIZED
				return derp.UnauthorizedError("middleware.Owner", "sterankoContext.Authorization", err)
			}

			// Guarantee that we have a model.Authorization
			if auth, ok := authorization.(*model.Authorization); ok {

				// If not the domain owner, return FORBIDDEN
				if !auth.DomainOwner {
					return derp.ForbiddenError("middleware.Owner", "authorization.DomainOwner", nil)
				}

				// Success!  Continue on to the next handler
				return next(ctx)
			}

			// This should never happen
			return derp.InternalError("middleware.Owner", "authorization.(*model.Authorization)", nil)
		}

		// This should never happen
		return derp.InternalError("middleware.Owner", "sterankoContext.Authorization", nil)
	}
}
