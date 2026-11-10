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

// OAuthAuthorizationRequest holds the parameters of an OAuth authorization request
//
// https://docs.joinmastodon.org/methods/oauth/#authorize
// GET /oauth/authorize
// Returns: Authorization code
type OAuthAuthorizationRequest struct {
	ResponseType        string `json:"responseType"        query:"response_type"         form:"response_type"`
	ClientID            string `json:"clientId"            query:"client_id"             form:"client_id"`
	RedirectURI         string `json:"redirectUri"         query:"redirect_uri"          form:"redirect_uri"`
	Scope               string `json:"scope"               query:"scope"                 form:"scope"`
	State               string `json:"state"               query:"state"                 form:"state"`
	CodeChallenge       string `json:"codeChallenge"       query:"code_challenge"        form:"code_challenge"`
	CodeChallengeMethod string `json:"codeChallengeMethod" query:"code_challenge_method" form:"code_challenge_method"`
	ForceLogin          bool   `json:"forceLogin"          query:"force_login"           form:"force_login"`
	Language            string `json:"language"            query:"language"              form:"language"`
	Me                  string `json:"me"                  query:"me"` // IndieAuth §5.2 hint, shown on the consent page and never posted back
}

// NewOAuthAuthorizationRequest returns a fully initialized, empty OAuthAuthorizationRequest
func NewOAuthAuthorizationRequest() OAuthAuthorizationRequest {
	return OAuthAuthorizationRequest{}
}

// Scopes returns the requested OAuth scopes, split into individual values
func (req OAuthAuthorizationRequest) Scopes() []string {
	return ParseOAuthScopes(req.Scope)
}

// hasScope returns TRUE if the request names at least one scope
func (req OAuthAuthorizationRequest) hasScope() bool {

	return len(req.Scopes()) > 0
}

// Validate confirms that a request is valid based on the settings in the OAuthClient.
// This method MAY update the request if certain values are missing.
func (req *OAuthAuthorizationRequest) Validate(client OAuthClient) error {

	const location = "model.OAuthAuthorizationRequest.Validate"

	// RULE: A missing redirect_uri defaults only for a registered client that lists one
	if err := req.defaultRedirectURI(client); err != nil {
		return err
	}

	// RULE: The redirect_uri must be one the client lists, or on a URL client's own origin
	if !client.AllowsRedirectURI(req.RedirectURI) {
		return derp.BadRequest(location, "Invalid redirect_uri", "provided: "+req.RedirectURI, "allowed: "+strings.Join(client.RedirectURIs, ","))
	}

	// RULE: A registered client that names no scope gets every scope it registered.  A URL client
	// that names none asks only who the User is (IndieAuth §5.2), so its scope stays empty.
	if (req.Scope == "") && (client.ClientURL == "") {
		req.Scope = strings.Join(client.Scopes, " ")
	}

	// RULE: Verify that scope is valid.  A request that names no scope asks for nothing, so there is nothing to refuse
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

	// RULE: The implicit grant returns an access token, so a request that names no scope has nothing to return
	if (req.ResponseType == "token") && !req.hasScope() {
		return derp.BadRequest(location, "A scope is required when response_type is 'token'")
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

// defaultRedirectURI fills in a missing redirect_uri with the first one a registered client lists,
// and refuses a missing one from a URL client, or from a client that lists none
func (req *OAuthAuthorizationRequest) defaultRedirectURI(client OAuthClient) error {

	const location = "model.OAuthAuthorizationRequest.defaultRedirectURI"

	if req.RedirectURI != "" {
		return nil
	}

	// RULE: A URL client must name its redirect_uri, because nothing is registered for it (IndieAuth §5.2)
	if client.ClientURL != "" {
		return derp.BadRequest(location, "redirect_uri is required")
	}

	// RULE: A client that lists no redirect_uri has nothing to default to
	if len(client.RedirectURIs) == 0 {
		return derp.BadRequest(location, "redirect_uri is required, because this client lists none")
	}

	req.RedirectURI = client.RedirectURIs[0]
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
