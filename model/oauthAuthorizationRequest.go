package model

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
)

/******************************************
 * OAuth API Methods
 * Generate and manage OAuth tokens
 * https://docs.joinmastodon.org/methods/oauth/
******************************************/

// https://docs.joinmastodon.org/methods/oauth/#authorize
// GET /oauth/authorize
// Returns: Authorization code
type OAuthAuthorizationRequest struct {
	ResponseType        string `query:"response_type"         form:"response_type"`
	ClientID            string `query:"client_id"             form:"client_id"`
	RedirectURI         string `query:"redirect_uri"          form:"redirect_uri"`
	Scope               string `query:"scope"                 form:"scope"`
	State               string `query:"state"                 form:"state"`
	CodeChallenge       string `query:"code_challenge"        form:"code_challenge"`
	CodeChallengeMethod string `query:"code_challenge_method" form:"code_challenge_method"`
	ForceLogin          bool   `query:"force_login"           form:"force_login"`
	Language            string `query:"language"              form:"language"`
}

// NewOAuthAuthorizationRequest returns a fully initialized, empty OAuthAuthorizationRequest
func NewOAuthAuthorizationRequest() OAuthAuthorizationRequest {
	return OAuthAuthorizationRequest{}
}

// Scopes returns the requested OAuth scopes, split into individual values
func (req OAuthAuthorizationRequest) Scopes() []string {
	return strings.Split(req.Scope, " ")
}

// Validate confirms that a request is valid based on the settings in the OAuthClient.
// This method MAY update the request if certain values are missing.
func (req *OAuthAuthorizationRequest) Validate(client OAuthClient) error {

	const location = "model.OAuthAuthorizationRequest.Validate"

	if len(client.RedirectURIs) == 0 {
		return derp.Internal(location, "Application must have at least one redirect_uri")
	}

	// RULE: If missing, use default value for RedirectURI
	if req.RedirectURI == "" {
		req.RedirectURI = client.RedirectURIs[0]
	}

	// RULE: Verify that redirect URI is valid
	if !slice.Contains(client.RedirectURIs, req.RedirectURI) {
		return derp.BadRequest(location, "Invalid redirect_uri", "provided: "+req.RedirectURI, "allowed: "+strings.Join(client.RedirectURIs, ","))
	}

	// RULE: If missing, use default value for Scope
	if req.Scope == "" {
		req.Scope = strings.Join(client.Scopes, " ")
	}

	// RULE: Verify that scope is valid
	if client.Scopes.NotEmpty() {
		for _, scope := range req.Scopes() {
			if !slice.Contains(client.Scopes, scope) {
				return derp.BadRequest(location, "Invalid scope", "provided: "+scope, "allowed: "+strings.Join(client.Scopes, ","))
			}
		}
	}

	// RULE: ResponseType must be one of the approved values.
	switch req.ResponseType {
	case "code":
	case "token":
	default:
		req.ResponseType = "code"
	}

	// RULE: PKCE (RFC 7636). Normalize and validate the challenge method.
	if err := req.validatePKCE(); err != nil {
		return err
	}

	// RULE: Public clients (no client_secret) MUST use PKCE for the code grant
	// (RFC 8252 / OAuth 2.1), so an intercepted authorization code is useless
	// without the matching code_verifier.  The implicit "token" grant has no code
	// to intercept, so it is exempt.
	if req.ResponseType == "code" && client.ClientSecret == "" && req.CodeChallenge == "" {
		return derp.BadRequest(location, "This client must use PKCE (a code_challenge is required)")
	}

	// Success
	return nil
}

// validatePKCE normalizes and validates the PKCE parameters (RFC 7636). If a
// code_challenge is present, the method defaults to "plain" when omitted (§4.3)
// and must be one this server supports.
func (req *OAuthAuthorizationRequest) validatePKCE() error {

	if req.CodeChallenge == "" {
		return nil
	}

	if req.CodeChallengeMethod == "" {
		req.CodeChallengeMethod = PKCEMethodPlain
	}

	if !IsValidPKCEMethod(req.CodeChallengeMethod) {
		return derp.BadRequest("model.OAuthAuthorizationRequest.validatePKCE", "Unsupported code_challenge_method (expected 'plain' or 'S256')", req.CodeChallengeMethod)
	}

	return nil
}
