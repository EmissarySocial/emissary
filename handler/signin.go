package handler

import (
	"bytes"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/server"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetSignIn generates an echo.HandlerFunc that handles GET /signin requests
func GetSignIn(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var buffer bytes.Buffer
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetSignin", "Error getting factory"))
		}

		template := factory.Layout().Template

		if err := template.ExecuteTemplate(&buffer, "signin", "error message goes here."); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetSignin", "Error executing template"))
		}

		return ctx.HTML(200, buffer.String())
	}
}

// PostSignIn generates an echo.HandlerFunc that handles POST /signin requests
func PostSignIn(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			ctx.Response().Header().Add("HX-Trigger", "SigninError")
			return ctx.HTML(200, postSigninError("Invalid Request.  Please try again later."))
		}

		s := factory.Steranko()

		var txn steranko.SigninTransaction

		if err := ctx.Bind(&txn); err != nil {
			ctx.Response().Header().Add("HX-Trigger", "SigninError")
			return ctx.HTML(200, postSigninError("Invalid Request.  Please try again later."))
		}

		result := s.Signin(txn)

		if result.Error != nil {
			return ctx.HTML(200, postSigninError(derp.Message(result.Error)))
		}

		// Set Cookies
		ctx.SetCookie(&http.Cookie{
			Name:     "Authorization",
			Value:    result.JWT,              // Set the cookie's value
			MaxAge:   63072000,                // Max-Age is 2 YEARS (60s * 60min * 24h * 365d * 2y)
			Path:     "/",                     // This allows the cookie on all paths of this site.
			Secure:   ctx.IsTLS(),             // Set secure cookies if we're on a secure connection
			HttpOnly: true,                    // Cookies should only be accessible via HTTPS (not client-side scripts)
			SameSite: http.SameSiteStrictMode, // Strict same-site policy prevents cookies from being used by other sites.
			// NOTE: Domain is excluded because it is less restrictive than omitting it. [https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies]
		})

		ctx.Response().Header().Add("HX-Trigger", "SigninSuccess")

		return ctx.NoContent(200)
	}
}

// PostSignOut generates an echo.HandlerFunc that handles POST /signout requests
func PostSignOut(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		ctx.SetCookie(&http.Cookie{
			Name:     cookieName(ctx),         // Get the Cookie name to use for this context.
			Value:    "",                      // Erase the value of the cookie
			MaxAge:   0,                       // Expires the cookie immediately
			Path:     "/",                     // This allows the cookie on all paths of this site.
			Secure:   ctx.IsTLS(),             // Set secure cookies if we're on a secure connection
			HttpOnly: true,                    // Cookies should only be accessible via HTTPS (not client-side scripts)
			SameSite: http.SameSiteStrictMode, // Strict same-site policy prevents cookies from being used by other sites.
		})

		return ctx.Redirect(http.StatusSeeOther, "/signin")
	}
}

func postSigninError(message string) string {
	return `<div class="uk-alert uk-alert-danger">` + message + `</div>`
}

func cookieName(ctx echo.Context) string {

	// If this is a secure domain...
	if ctx.IsTLS() {
		// Use a cookie name that can only be set on an SSL connection, and is "domain-locked"
		// [https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#cookie_prefixes]
		return "__Host-Authorization"
	}

	return "Authorization"
}
