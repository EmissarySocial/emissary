package middleware

import (
	"net/http"

	"github.com/benpate/domain"
	domaintools "github.com/benpate/domain"
	"github.com/labstack/echo/v4"
)

func HttpsRedirect(handler echo.HandlerFunc) echo.HandlerFunc {

	return func(context echo.Context) error {

		// If this is already an HTTPS request, then continue
		if context.Scheme() == "https" {
			return handler(context)
		}

		request := context.Request()

		// Do not HTTPS for localhost
		hostname := domaintools.Hostname(request)
		if domain.IsLocalhost(hostname) {
			return handler(context)
		}

		// Otherwise, permanently redirect all other requests to HTTPS
		request.URL.Scheme = "https"

		return context.Redirect(http.StatusPermanentRedirect, request.URL.String())
	}
}
