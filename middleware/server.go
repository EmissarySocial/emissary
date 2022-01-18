package middleware

import (
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
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
				return derp.NewNotFoundError("whisper", "Not Found")
			}

			if !domain.ShowAdmin {
				return derp.NewNotFoundError("whisper", "Not Found")
			}

			return next(ctx)
		}
	}
}
