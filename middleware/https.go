package middleware

import (
	"net/http"

	"github.com/benpate/domain"
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
		if domain.IsLocalhost(request.Host) {
			return handler(context)
		}

		request.URL.Scheme = "https"

		// Permanently redirect all other requests to HTTPS endpoint
		return context.Redirect(http.StatusTemporaryRedirect, request.URL.String())
	}
}
