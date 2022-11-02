package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// GetOAuth looks up the OAuth provider and forwards to the appropriate endpoint
func GetOAuth(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.GetOAuth"

	return func(ctx echo.Context) error {

		// Try to look up the factory for this domain
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Could not get factory", ctx.Request().URL.String())
		}

		// Get domain service and redirect URL
		providerID := ctx.Param("provider")
		domainService := factory.Domain()
		redirectURL, err := domainService.OAuthCodeURL(providerID)

		if err != nil {
			return derp.Wrap(err, location, "Could not get redirect URL", providerID)
		}

		// Success!!
		return ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}

func GetOAuthCallback(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.OAuthCallback"

	return func(ctx echo.Context) error {

		// Try to look up the factory for this domain
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Could not get factory", ctx.Request().URL.String())
		}

		// Try to ge the current domain
		domainService := factory.Domain()

		providerID := ctx.Param("provider")
		code := ctx.QueryParam("code")
		state := ctx.QueryParam("state")

		if err := domainService.OAuthExchange(providerID, state, code); err != nil {
			return derp.Wrap(err, location, "Error exchanging code for token", providerID, code)
		}

		return ctx.Redirect(http.StatusTemporaryRedirect, "/admin/connections")
	}
}

func OAuthRedirect(factory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return nil
	}
}
