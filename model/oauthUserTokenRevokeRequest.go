package model

import "github.com/benpate/derp"

// https://docs.joinmastodon.org/methods/oauth/#revoke
// POST /oauth/revoke
// Returns: Empty struct
// Revoke an access token to make it no longer valid for use
type OAuthUserTokenRevokeRequest struct {
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
	Token        string `form:"token"`
}

func NewOAuthUserTokenRevokeRequest() OAuthUserTokenRevokeRequest {
	return OAuthUserTokenRevokeRequest{}
}

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
