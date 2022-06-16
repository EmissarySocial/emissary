package middleware

import (
	"net/http"
	"strings"

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
		if request.Host == "localhost" {
			return handler(context)
		}

		// Do not use HTTPS for *.local
		if strings.HasSuffix(request.Host, ".local") {
			return handler(context)
		}

		// Permanently redirect all other requests to HTTPS endpoint
		return context.Redirect(http.StatusTemporaryRedirect, "https://"+request.Host+request.URL.Path)
	}
}
