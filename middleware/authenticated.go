package middleware

/* REMOVED BECAUSE MIDDLEWARE NO LONGER WORKS WITH STERANKO CONTEXTS

// Authenticated middleware guarantees that the request is being performed by a website owner
func Authenticated(next echo.HandlerFunc) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Guarantee that we have a steranko.Context
		if sterankoContext, ok := ctx.(*steranko.Context); ok {

			// If not authorized, return NOT AUTHORIZED
			if _, err := sterankoContext.Authorization(); err != nil {
				return derp.UnauthorizedError("middleware.Authenticated", "Authorization must be sterankoContext.Authorization()", err)
			}

			// Success
			return next(ctx)
		}

		// This should never happen
		return derp.InternalError("middleware.Authenticated", "Context must be ctx.(*steranko.Context)", nil)
	}
}
*/
