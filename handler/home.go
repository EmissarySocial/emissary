package handler

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/labstack/echo/v4"
)

func GetHome(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetHome"

	return func(ctx echo.Context) error {

		// If this is a valid domain, then redirect to the user's home page. (Handles 99% of requests)
		if factory, err := serverFactory.ByContext(ctx); err == nil {

			// Load the domain from the memory cache and find the forwarding URL
			domain := factory.Domain().Get()
			authorization := getAuthorization(ctx)
			homePage := domain.DefaultPage(authorization)

			// Redirect the user to the appropriate home page
			return ctx.Redirect(http.StatusTemporaryRedirect, homePage)
		}

		// Otherwise, look up the hostname to see if this is a personalized domain (Like: yomama.serer.social)
		hostname := dt.TrueHostname(ctx.Request())
		username, hostname, exists := strings.Cut(hostname, ".")

		if !exists {
			return derp.MisdirectedRequest(location, "Username/hostname not found")
		}

		if _, err := serverFactory.ByHostname(hostname); err == nil {
			redirectTo := dt.AddProtocol(hostname) + "/@" + username
			return ctx.Redirect(http.StatusTemporaryRedirect, redirectTo)
		}

		// Fall through means unrecognized domain. Return a 404 error.
		return derp.MisdirectedRequest(location, "Hostname not found")
	}
}
