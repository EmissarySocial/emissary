package steranko

import (
	"github.com/labstack/echo/v4"
)

// Middleware wraps the original echo context with the Steranko context.
func (s *Steranko) Middleware(next echo.HandlerFunc) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Verify that the request is valid
		if err := s.ApproveRequest(ctx); err != nil {
			return err
		}

		return next(&Context{
			Context:  ctx,
			steranko: s,
		})
	}
}

// Middleware is a standalone middleware that works for multi-tenant
// environments, where you may need to use a factory to load the specific
// steranko settings depending on the domain being called.
func Middleware(factory Factory) echo.MiddlewareFunc {

	// this is the middleware function
	return func(next echo.HandlerFunc) echo.HandlerFunc {

		// this handles the specific request
		return func(ctx echo.Context) error {

			// find the correct steranko instance
			s, err := factory.Steranko(ctx)

			// handle errors (if necessary)
			if err != nil {
				return err
			}

			// Verify that the request is valid
			if err := s.ApproveRequest(ctx); err != nil {
				return err
			}

			// call the next function in the chain, now
			// using a Steranko context instead of the original
			return next(&Context{
				Context:  ctx,
				steranko: s,
			})
		}
	}
}
