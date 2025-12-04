package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"golang.org/x/oauth2"
)

/******************************************
 * OAuth Handshake Methods
 ******************************************/

// OAuthExchange trades a temporary OAuth code for a valid OAuth token
func (service *Import) OAuthExchange(session data.Session, record *model.Import, state string, code string) error {

	const location = "service.Import.OAuthExchange"

	// Validate the state across requests
	if state != record.ImportID.Hex() {
		return derp.BadRequestError(location, "OAuth State must match internal records")
	}

	// Try to generate the OAuth token
	token, err := record.OAuthConfig.Exchange(session.Context(), code,
		oauth2.SetAuthURLParam("code_verifier", string(record.OAuthChallenge)),
		oauth2.SetAuthURLParam("redirect_uri", service.OAuthClientCallbackURL()))

	if err != nil {
		return derp.Wrap(err, location, "Unable to exchange OAuth code for token", derp.WithInternalError())
	}

	// Update the record with the new OAuth token and "Authorized" status
	record.StateID = model.ImportStateAuthorized
	record.OAuthToken = token

	// Save the Import record
	if service.Save(session, record, "OAuth Exchange") != nil {
		return derp.InternalError(location, "Unable to save domain")
	}

	// Success!
	return nil
}

// GetAuthToken retrieves the OAuth token for the specified provider.  If the token has expired
// then it is refreshed (and saved) automatically before returning.
func (service *Import) GetOAuthToken(session data.Session, record *model.Import) (*oauth2.Token, error) {

	const location = "service.Import.GetOAuthToken"

	// RULE: We must have an existing token to start
	if record.OAuthToken == nil {
		return nil, derp.BadRequestError(location, "No OAuth token found.  This should never happen.")
	}

	// Use TokenSource to update tokens when they expire.
	source := record.OAuthConfig.TokenSource(session.Context(), record.OAuthToken)

	newToken, err := source.Token()

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to refresh OAuth token")
	}

	// If the token has changed, save it
	if record.OAuthToken.AccessToken != newToken.AccessToken {
		record.OAuthToken = newToken
		if err := service.Save(session, record, "Refreshing OAuth Token"); err != nil {
			return nil, derp.Wrap(err, location, "Unable to save refreshed Token")
		}
	}

	// Success!
	return newToken, nil
}

// OAuthClientCallbackURL returns the specific callback URL to use for this host and provider.
func (service *Import) OAuthClientCallbackURL() string {
	return service.host + "/oauth/clients/import/callback"
}
