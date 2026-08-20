package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/cimd"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/sliceof"
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

// GetOAuthCallback receives the OAuth callback from a third-party provider, and exchanges the code for a token
func GetOAuthCallback(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.OAuthCallback"

	// Try to ge the current domain
	domainService := factory.Domain()

	providerID := ctx.Param("provider")
	code := ctx.QueryParam("code")
	state := ctx.QueryParam("state")

	if err := domainService.OAuthExchange(session, providerID, state, code); err != nil {
		return derp.Wrap(err, location, "Exchanging code for token", providerID, code)
	}

	return ctx.Redirect(http.StatusTemporaryRedirect, "/admin/connections")
}

// GetOAuthClientMetadata returns OAuth client metadata for this domain
// https://client.dev
func GetOAuthClientMetadata(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	domain := factory.Domain().Get()

	metadata := cimd.Metadata{
		ClientID:   domain.Host() + "/oauth/metadata",
		ClientName: domain.Label,
		ClientURI:  domain.Host(),
		LogoURI:    domain.IconURL(),
		RedirectURIs: sliceof.String{
			domain.Host() + "/oauth/clients/import/callback",
		},
		GrantTypes: sliceof.String{
			"authorization_code",
		},
	}

	// Success!!
	return ctx.JSON(http.StatusOK, metadata)
}

// OAuthRedirect is the OAuth redirect endpoint, which is registered but does nothing yet
func OAuthRedirect(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	return nil
}
