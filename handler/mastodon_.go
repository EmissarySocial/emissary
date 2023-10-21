package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/golang-jwt/jwt/v5"
)

func Mastodon(serverFactory *server.Factory) toot.API {

	return toot.API{
		Authorize:                        mastodon_Authorize(serverFactory),
		PostApplication:                  mastodon_PostApplication(serverFactory),
		GetApplication_VerifyCredentials: mastodon_GetApplication_VerifyCredentials(serverFactory),

		PostStatus: mastodon_PostStatus(serverFactory),
		GetStatus:  mastodon_GetStatus(serverFactory),
	}
}

func mastodon_Authorize(serverFactory *server.Factory) func(*http.Request, ...string) bool {

	return func(request *http.Request, scopes ...string) bool {
		return true
	}
}

func getMastodonAuthorization(tokenString string) (model.Authorization, error) {

	const location = "handler.getMastodonAuthorization"

	result := model.NewAuthorization()

	// Parse it as a JWT token
	// TODO: CRITICAL: Add WithValidateMthods() to this call.
	token, err := jwt.ParseWithClaims(tokenString, result, nil)

	if err != nil {
		return result, derp.Wrap(err, location, "Error parsing token")
	}

	if !token.Valid {
		return result, derp.NewForbiddenError(location, "Invalid token: Invalid JWT")
	}

	if result.UserID.IsZero() {
		return model.Authorization{}, derp.NewForbiddenError(location, "Invalid token: missing UserID")
	}

	if result.ClientID.IsZero() {
		return result, derp.NewForbiddenError(location, "Token must be an OAuth token, not a user token")
	}

	return result, nil
}
