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
	ResponseType string `query:"response_type" form:"response_type"`
	ClientID     string `query:"client_id"     form:"client_id"`
	RedirectURI  string `query:"redirect_uri"  form:"redirect_uri"`
	Scope        string `query:"scope"         form:"scope"`
	ForceLogin   bool   `query:"force_login"   form:"force_login"`
	Language     string `query:"language"      form:"language"`
}

func NewOAuthAuthorizationRequest() OAuthAuthorizationRequest {
	return OAuthAuthorizationRequest{}
}

func (req OAuthAuthorizationRequest) Scopes() []string {
	return strings.Split(req.Scope, " ")
}

// Validate confirms that a request is valid based on the settings in the OAuthApplication.
// This method MAY update the request if certain values are missing.
func (req *OAuthAuthorizationRequest) Validate(app OAuthApplication) error {

	const location = "model.OAuthAuthorizationRequest.Validate"

	if len(app.RedirectURIs) == 0 {
		return derp.NewInternalError(location, "Application must have at least one redirect_uri")
	}

	// RULE: If missing, use default value for RedirectURI
	if req.RedirectURI == "" {
		req.RedirectURI = app.RedirectURIs[0]

	}

	// RULE: Verify that redirect URI is valid
	if !slice.Contains(app.RedirectURIs, req.RedirectURI) {
		return derp.NewBadRequestError(location, "Invalid redirect_uri", req.RedirectURI)
	}

	// RULE: If missing, use default value for Scope
	if req.Scope == "" {
		req.Scope = strings.Join(app.Scopes, " ")
	}

	// RULE: Verify that scope is valid
	for _, scope := range req.Scopes() {
		if !slice.Contains(app.Scopes, scope) {
			return derp.NewBadRequestError(location, "Invalid scope", scope)
		}
	}

	// RULE: ResponseType must be one of the approved values.
	switch req.ResponseType {
	case "code":
	case "token":
	default:
		req.ResponseType = "code"
	}

	// Success
	return nil
}
