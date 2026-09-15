package model

import "github.com/benpate/derp"

// OAuthUserTokenRevokeRequest holds the parameters of an OAuth token revocation request
//
// https://docs.joinmastodon.org/methods/oauth/#revoke
// POST /oauth/revoke
// Returns: Empty struct
// Revoke an access token to make it no longer valid for use
type OAuthUserTokenRevokeRequest struct {
	ClientID     string `json:"clientId" form:"client_id"`
	ClientSecret string `json:"clientSecret" form:"client_secret"`
	Token        string `json:"token" form:"token"`
}

// NewOAuthUserTokenRevokeRequest returns a fully initialized, empty OAuthUserTokenRevokeRequest
func NewOAuthUserTokenRevokeRequest() OAuthUserTokenRevokeRequest {
	return OAuthUserTokenRevokeRequest{}
}

// Validate confirms that this revoke request presents the credentials of the provided OAuthClient
func (req *OAuthUserTokenRevokeRequest) Validate(app OAuthClient) error {

	const location = "model.OAuthUserTokenRevokeRequest.Validate"

	if req.ClientID != app.ClientID.Hex() {
		return derp.BadRequest(location, "Invalid client_iD")
	}

	if req.ClientSecret != app.ClientSecret {
		return derp.BadRequest(location, "Invalid client_secret")
	}

	return nil
}
