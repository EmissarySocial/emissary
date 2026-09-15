package model

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
)

// OAuthUserTokenRequest holds the parameters of an OAuth token request
//
// https://docs.joinmastodon.org/methods/oauth/#token
// POST /oauth/token
// Returns: Token
// Obtain an access token, to be used during API calls that are not public
type OAuthUserTokenRequest struct {
	GrantType    string `json:"grantType"    form:"grant_type"`
	Code         string `json:"code"         form:"code"`
	RefreshToken string `json:"refreshToken" form:"refresh_token"` // The rotating refresh token, presented for a refresh_token grant (RFC 6749 §6)
	ClientID     string `json:"clientId"     form:"client_id"`
	ClientSecret string `json:"clientSecret" form:"client_secret"`
	RedirectURI  string `json:"redirectUri"  form:"redirect_uri"`
	Scope        string `json:"scope"        form:"scope"`
	CodeVerifier string `json:"codeVerifier" form:"code_verifier"` // PKCE (RFC 7636) verifier, presented to redeem a code bound to a code_challenge
}

// NewOAuthUserTokenRequest returns a fully initialized, empty OAuthUserTokenRequest
func NewOAuthUserTokenRequest() OAuthUserTokenRequest {
	return OAuthUserTokenRequest{}
}

// Scopes returns the requested OAuth scopes, split into individual values, defaulting to "read"
func (o OAuthUserTokenRequest) Scopes() []string {
	if o.Scope == "" {
		return []string{"read"}
	}

	scope := strings.ReplaceAll(o.Scope, ",", " ")
	return strings.Split(scope, " ")
}

// Validate confirms that a request is valid based on the settings in the OAuthClient.
// This method MAY update the request if certain values are missing.
func (req *OAuthUserTokenRequest) Validate(client OAuthClient) error {

	const location = "model.OAuthUserTokenRequest.Validate"

	// RULE: ClientID must match the client application

	if notOneOf(req.ClientID, client.ClientURL, client.ClientID.Hex()) {
		return derp.BadRequest(location, "Invalid client_id", client, req)
	}

	// RULE: ClientSecret must match the client application
	if req.ClientSecret != client.ClientSecret {
		return derp.BadRequest(location, "Invalid client_secret", client, req)
	}

	// RULE: Client must have at least one redirect_uri
	if len(client.RedirectURIs) == 0 {
		return derp.Internal(location, "Client must have at least one redirect_uri")
	}

	// RULE: If missing, use default value for RedirectURI
	if req.RedirectURI == "" {
		req.RedirectURI = client.RedirectURIs[0]
	}

	// RULE: Verify that redirect URI is valid
	if !slice.Contains(client.RedirectURIs, req.RedirectURI) {
		return derp.BadRequest(location, "Invalid redirect_uri", client, req)
	}

	// RULE: If missing, use default value for Scope
	if req.Scope == "" {
		req.Scope = strings.Join(client.Scopes, " ")
	}

	// RULE: Verify that scope is valid
	for _, scope := range req.Scopes() {
		if !slice.Contains(client.Scopes, scope) {
			return derp.BadRequest(location, "Invalid scope", scope)
		}
	}

	// RULE: ResponseType must be one of the approved values.
	switch req.GrantType {
	case "code":
	case "token":
	default:
		req.GrantType = "code"
	}

	// Success
	return nil
}
