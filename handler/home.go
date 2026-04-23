package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/steranko"
)

// GetHome handles requests to the root URL ("/") and redirects to the appropriate home page
// based on the user's authentication status and domain settings.
func GetHome(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	// Load the domain from the memory cache and find the forwarding URL
	domain := factory.Domain().Get()
	authorization := getAuthorization(ctx)
	homePage := domain.DefaultPage(authorization)

	// Redirect the user to the appropriate home page
	return ctx.Redirect(http.StatusTemporaryRedirect, homePage)
}
