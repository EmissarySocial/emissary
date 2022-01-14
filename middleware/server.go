package middleware

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

// ServerAdmin generates a middleware that enforces security permissions
// on the /server directory.  It confirms that uses the server's config file
// allows access to this directory and validates the client's JWT cookie
func ServerAdmin(factoryManager *server.FactoryManager) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			domain, err := factoryManager.DomainByName(ctx.Request().Host)

			if err != nil {
				derp.Report(err)
				return derp.NewNotFoundError("ghost", "Not Found")
			}

			if !domain.ShowAdmin {
				return derp.NewNotFoundError("ghost", "Not Found")
			}

			return next(ctx)
		}
	}
}
