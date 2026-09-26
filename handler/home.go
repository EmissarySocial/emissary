package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// GetHome redirects a visitor to the home page configured for this Domain
func GetHome(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetHome"

	return func(ctx echo.Context) error {

		// If this is a valid domain, then redirect to the user's home page. (Handles 99% of requests)
		if factory, err := serverFactory.ByContext(ctx); err == nil {

			// Read the cached Domain record and find the forwarding URL
			readOnlyDomain := factory.Domain().Cached()
			authorization := getAuthorization(ctx)
			homePage := readOnlyDomain.DefaultPage(authorization)

			// Redirect the user to the appropriate home page
			return ctx.Redirect(http.StatusTemporaryRedirect, homePage)
		}

		// Otherwise, look up the hostname to see if this is a personalized domain (Like: yomama.server.social)
		hostname := serverFactory.Hostname(ctx.Request())
		parentFactory, username, err := serverFactory.ByPersonalizedHostname(hostname)

		if err != nil {
			return derp.Wrap(err, location, "Hostname not found", "hostname: "+hostname)
		}

		// RULE: Forward ONLY to a User that exists.  The parent hostname alone cannot tell a
		// personalized domain from a subdomain that used to be a Domain of its own, so a redirect
		// issued without this reports an outage to the visitor as somebody else's 404 (BUG-142)
		session, err := parentFactory.Server().Session(ctx.Request().Context())

		if err != nil {
			return derp.Wrap(err, location, "Opening database session", "hostname: "+hostname)
		}

		defer session.Close()

		user := model.NewUser()

		if err := parentFactory.User().LoadByUsername(session, username, &user); err != nil {
			return derp.MisdirectedRequest(location, "Hostname not found", "hostname: "+hostname)
		}

		// Forward the visitor to this User's profile on the parent Domain
		return ctx.Redirect(http.StatusTemporaryRedirect, parentFactory.Host()+"/@"+username)
	}
}
