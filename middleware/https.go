package middleware

import (
	"net/http"

	"github.com/EmissarySocial/emissary/tools/domain"
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

		// Permanently redirect all other requests to HTTPS endpoint
		return context.Redirect(http.StatusTemporaryRedirect, "https://"+request.Host+request.URL.Path)
	}
}
