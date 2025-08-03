package middleware

import (
	"net/http"

	dt "github.com/benpate/domain"
	"github.com/labstack/echo/v4"
)

// HttpsRedirect is a middleware that redirects all HTTP requests to HTTPS
// when the request comes from a public network.
func HttpsRedirect(handler echo.HandlerFunc) echo.HandlerFunc {

	return func(context echo.Context) error {

		// If this is already an HTTPS request, then continue
		if context.Scheme() == "https" {
			return handler(context)
		}

		request := context.Request()

		// Do not require HTTPS for localhost
		// This is okay for local domains (even behind a proxy) because
		// unencrypted traffic will only be on the private network.
		if dt.IsLocalhost(request.Host) {
			return handler(context)
		}

		// Otherwise, permanently redirect all other requests to HTTPS
		request.URL.Scheme = "https"

		return context.Redirect(http.StatusPermanentRedirect, request.URL.String())
	}
}
