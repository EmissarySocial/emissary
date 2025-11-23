package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetOAuth looks up the OAuth provider and forwards to the appropriate endpoint
func GetOAuth(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetOAuth"

	// Get domain service and redirect URL
	providerID := ctx.Param("provider")
	redirectURL, err := factory.Domain().OAuthCodeURL(session, providerID)

	if err != nil {
		return derp.Wrap(err, location, "Could not get redirect URL", providerID)
	}

	// Success!!
	return ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func GetOAuthCallback(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.OAuthCallback"

	// Try to ge the current domain
	domainService := factory.Domain()

	providerID := ctx.Param("provider")
	code := ctx.QueryParam("code")
	state := ctx.QueryParam("state")

	if err := domainService.OAuthExchange(session, providerID, state, code); err != nil {
		return derp.Wrap(err, location, "Unable to exchange code for token", providerID, code)
	}

	return ctx.Redirect(http.StatusTemporaryRedirect, "/admin/connections")
}

func OAuthRedirect(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	return nil
}
