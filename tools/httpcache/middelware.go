package httpcache

import "github.com/labstack/echo/v4"

type Middleware struct {
	cache HTTPCache
}

// Middleware implements an echo middleware that can be used to cache outbound HTTP responses.
func (middleware *Middleware) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return nil
	}
}
