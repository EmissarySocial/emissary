package middleware

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// ServerAdminExists generates a middleware that enforces security permissions
// on the /server directory.  It confirms that uses the server's config file
// allows access to this directory
func ServerAdminExists(factory *server.Factory) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			domain, err := factory.DomainByName(ctx.Request().Host)

			if err != nil {
				return derp.NewNotFoundError("whisperverse.middleware.ServerAdmin", "Not Found")
			}

			if !domain.ShowAdmin {
				return derp.NewNotFoundError("whisperverse.middleware.ServerAdmin", "Not Found")
			}

			return next(ctx)
		}
	}
}

// ServerAdminAllowed generates a middleware that enforces security permissions
// on the /server directory.  It validates the user's password for the admin site
func ServerAdminAllowed(factory *server.Factory) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			// Verify the admin password cookie
			// TODO: hash this value?
			password, err := ctx.Cookie("admin")

			if err != nil {
				return derp.NewForbiddenError("whisperverse.middleware.ServerAdmin", "Unauthorized")
			}

			if !factory.IsAdminPassword(password.Value) {
				return derp.NewForbiddenError("whisperverse.middleware.ServerAdmin", "Unauthorized")
			}

			return next(ctx)
		}
	}
}

func ServerAdminLogin(factory *server.Factory) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			if err := next(ctx); err != nil {

				b := html.New()
				b.H1().InnerHTML("Whisperverse")
				b.H2().InnerHTML("Enter Admin Password")
				return ctx.HTML(http.StatusOK, b.String())
			}

			return nil
		}
	}
}
