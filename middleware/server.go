package middleware

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/labstack/echo/v4"
)

// ServerAdmin generates a middleware that enforces security permissions
// on the /server directory.  It confirms that uses the server's config file
// allows access to this directory and that the user has an apporopriate
// password cookie set.  If not, then a signin page is displayed instead of
// of the original request
func ServerAdmin(factory *server.Factory) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			// Verify that the admin console is available on this domain
			domain, err := factory.DomainByName(ctx.Request().Host)

			if err != nil {
				return derp.NewNotFoundError("middleware.ServerAdmin", "Not Found")
			}

			if !domain.ShowAdmin {
				return derp.NewNotFoundError("middleware.ServerAdmin", "Not Found")
			}

			// Verify the admin password cookie
			password, err := ctx.Cookie("admin")

			// If the request includes valid cookie, then allow it to continue
			if (err == nil) && (factory.IsAdminPassword(password.Value)) {
				return next(ctx)
			}

			/** Fall through means we're handling the signin form **/

			// default error messages
			var errorMessage string

			// If this is a POST, then try to validate the password from the request Body
			if ctx.Request().Method == http.MethodPost {

				postData := make(map[string]string)

				if err := ctx.Bind(&postData); err == nil {

					// If the password is valid...
					if factory.IsAdminPassword(postData["password"]) {

						// ... set a cookie ...
						ctx.SetCookie(&(http.Cookie{
							Name:     "admin",
							Value:    factory.HashedPassword(),
							SameSite: http.SameSiteStrictMode,
							HttpOnly: true,
							// TODO: robusticate this cookie config
							// Expiration: ??
						}))

						// ... and redirect to the admin console.
						return ctx.Redirect(http.StatusTemporaryRedirect, factory.AdminURL())
					}
				}

				// Save error statuses
				errorMessage = "Invalid Signin"
			}

			// Otherwise, generate a signin form
			b := html.New()
			b.H1().InnerHTML("Whisperverse").Close()
			b.H2().InnerHTML("Enter Server Console Password").Close()

			if errorMessage != "" {
				b.Div().Style("color:red").InnerHTML(errorMessage).Close()
			}

			b.Form("post", factory.AdminURL())
			b.Input("password", "password").Style("width:400px", "font-size:14pt").Close()
			b.Button().Style("font-size:14pt").InnerHTML("Sign In").Close()

			return ctx.HTML(http.StatusOK, b.String())
		}
	}
}
