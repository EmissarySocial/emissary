package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetOAuthWellKnown serves the OAuth authorization-server metadata document (RFC 8414)
func GetOAuthWellKnown(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	// Get the server's host (scheme + host)
	host := factory.Host()

	// Build the response
	response := map[string]any{
		"issuer":                                host,
		"authorization_endpoint":                host + "/oauth/authorize",
		"token_endpoint":                        host + "/oauth/token",
		"revocation_endpoint":                   host + "/oauth/revoke",
		"response_types_supported":              []string{"code", "token"},
		"response_modes_supported":              []string{"query", "fragment"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"plain", "S256"}, // PKCE (RFC 7636)
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic"},
		"client_id_metadata_document_supported": true, // https://datatracker.ietf.org/doc/draft-ietf-oauth-client-id-metadata-document/
		"activitypub_object_id_as_client_id":    true, // https://w3id.org/fep/d8c2
		// "scopes_supported":       []string{"read", "write"},
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetOAuthAuthorization renders the "authorize this application" consent page
func GetOAuthAuthorization(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetOAuthAuthorization"

	// Collect the query parameters
	transaction := model.NewOAuthAuthorizationRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Binding query parameters")
	}

	// Load the OAuth Builder
	builder, err := build.NewOAuthAuthorization(factory, session, ctx.Request(), ctx.Response(), transaction, user)

	if err != nil {
		return derp.Wrap(err, location, "Generating Builder")
	}

	// Render the template
	return executeThemeTemplate(ctx, factory, "oauth", builder)
}

// PostOAuthAuthorization accepts the consent form, and issues a code or token per the requested grant type
func PostOAuthAuthorization(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.PostOAuthAuthorization"

	// Collect Form parameters
	transaction := model.NewOAuthAuthorizationRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Invalid form parameters")
	}

	// Convert the ClientID
	clientID, err := primitive.ObjectIDFromHex(transaction.ClientID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid client_id")
	}

	// Get Authorization
	authorization := getAuthorization(ctx)

	// Get Application
	clientService := factory.OAuthClient()
	application := model.NewOAuthClient()

	if err := clientService.LoadByClientID(session, clientID, &application); err != nil {
		return derp.Wrap(err, location, "Loading OAuth Application")
	}

	// Validate the transaction
	if err := transaction.Validate(application); err != nil {
		return derp.Wrap(err, location, "Invalid transaction")
	}

	// Create a UserToken
	userTokenService := factory.OAuthUserToken()
	userToken, err := userTokenService.Create(session, application, authorization, transaction)

	if err != nil {
		return derp.Wrap(err, location, "Creating OAuthUserToken")
	}

	// Complete the transaction based on the grant type
	switch transaction.ResponseType {

	case "code":
		return postOAuthAuthorization_code(ctx, userToken, transaction)

	case "token":
		return postOAuthAuthorization_token(ctx, userToken, transaction)
	}

	return derp.BadRequest(location, "Invalid response type", transaction.ResponseType)
}

// postOAuthAuthorization_code handles `code` grant types used by server authentication flow
func postOAuthAuthorization_code(ctx echo.Context, userToken model.OAuthUserToken, transaction model.OAuthAuthorizationRequest) error {

	// If this magic value is passed as the redirect URI, then we return the
	// authorization code in the <title> tag of the HTML for the user to copy into
	// their client, which then exchanges it at /oauth/token.
	// https://docs.joinmastodon.org/methods/apps/#form-data-parameters
	if transaction.RedirectURI == "urn:ietf:wg:oauth:2.0:oob" {
		b := html.New()
		b.HTML()
		b.Head()
		b.Title(userToken.Code())

		return ctx.HTML(http.StatusOK, b.String())
	}

	// Otherwise, start building the REAL redirect URI (using the query string)
	redirectURI, err := url.Parse(transaction.RedirectURI)

	if err != nil {
		return derp.Wrap(err, "handler.postOAuthAuthorization_code", "Invalid redirect_uri", transaction.RedirectURI)
	}

	// Add the code to the URI
	queryString := redirectURI.Query()
	queryString.Set("code", userToken.Code())
	queryString.Set("state", transaction.State)
	redirectURI.RawQuery = queryString.Encode()

	return ctx.Redirect(http.StatusFound, redirectURI.String())
}

// postOAuthAuthorization_code handles `token` grant types used by the client-side authentication flow
func postOAuthAuthorization_token(ctx echo.Context, userToken model.OAuthUserToken, transaction model.OAuthAuthorizationRequest) error {

	const location = "handler.postOAuthAuthorization_token"

	// If this magic value is passed as the redirect URI, then we just return the token in the <title> tag of the HTML
	// https://docs.joinmastodon.org/methods/apps/#form-data-parameters
	if transaction.RedirectURI == "urn:ietf:wg:oauth:2.0:oob" {
		b := html.New()
		b.HTML()
		b.Head()
		b.Title(userToken.Token)

		return ctx.HTML(http.StatusOK, b.String())
	}

	// Otherwise, start building the REAL redirect URI (using the hash fragment)
	redirectURI, err := url.Parse(transaction.RedirectURI)

	if err != nil {
		return derp.Wrap(err, location, "Parsing redirect_uri")
	}

	redirectURI.Fragment = "access_token=" + userToken.Token +
		"&state=" + transaction.State +
		"&token_type=Bearer"

	// Otherwise, we redirect to the redirect_uri
	return ctx.Redirect(http.StatusFound, redirectURI.String())
}

// PostOAuthToken handles the OAuth token endpoint (RFC 6749 §3.2), dispatching on
// grant_type. The authorization_code grant exchanges a code for an access+refresh
// pair; the refresh_token grant rotates a refresh token for a fresh pair.
func PostOAuthToken(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostOAuthToken"

	// Collect transaction data
	transaction := model.NewOAuthUserTokenRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Invalid form parameters")
	}

	// Load the OAuth Client (from ID or from ActorID)
	oauthClientService := factory.OAuthClient()
	oauthClient := model.NewOAuthClient()

	if err := oauthClientService.LoadByToken(session, transaction.ClientID, &oauthClient); err != nil {
		return derp.Wrap(err, location, "Invalid client_id", transaction)
	}

	// RULE: Dispatch on grant_type. An empty grant_type is treated as
	// authorization_code for backward compatibility with existing clients.
	switch transaction.GrantType {

	case "", "authorization_code":
		return postOAuthToken_authorizationCode(ctx, factory, session, oauthClient, transaction)

	case "refresh_token":
		return postOAuthToken_refresh(ctx, factory, session, oauthClient, transaction)
	}

	return derp.BadRequest(location, "Unsupported grant_type", transaction.GrantType)
}

// postOAuthToken_authorizationCode handles the authorization_code grant: it loads
// the grant identified by the code, authenticates the client (secret for a
// confidential client, PKCE for a public one), then exchanges the single-use code
// for an access+refresh pair.
func postOAuthToken_authorizationCode(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthClient model.OAuthClient, transaction model.OAuthUserTokenRequest) error {

	const location = "handler.postOAuthToken_authorizationCode"

	// Convert transaction.Code => grant ID
	userTokenID, err := primitive.ObjectIDFromHex(transaction.Code)

	if err != nil {
		return derp.Wrap(err, location, "Invalid code", transaction)
	}

	// Load the grant bound to this code. This performs NO authentication -- the
	// redemption is authenticated below, keyed on the client type.
	userTokenService := factory.OAuthUserToken()
	userToken := model.NewOAuthUserToken()

	if err := userTokenService.LoadByClientAndID(session, userTokenID, oauthClient.ClientID, &userToken); err != nil {
		return derp.Wrap(err, location, "Loading OAuthUserToken", transaction)
	}

	// RULE: Authenticate the code redemption (RFC 8252 / OAuth 2.1).  A
	// confidential client must present its matching client_secret; a public client
	// (no secret -- e.g. a CIMD/FEP-d8c2 client) must instead have bound its code
	// to a PKCE challenge, so an intercepted code is useless without the verifier.
	if oauthClient.IsConfidential() {
		if err := oauthClient.ValidateSecret(transaction.ClientSecret); err != nil {
			return derp.Wrap(err, location, "Invalid client_secret", transaction)
		}
	} else if !userToken.HasPKCEChallenge() {
		return derp.BadRequest(location, "This client must use PKCE (a code_verifier is required)", transaction)
	}

	// RULE: PKCE (RFC 7636). If the code was issued with a code_challenge, a
	// matching code_verifier is required to redeem it.
	if err := userToken.VerifyPKCE(transaction.CodeVerifier); err != nil {
		return derp.Wrap(err, location, "Invalid PKCE code_verifier")
	}

	// Exchange the single-use code for an access+refresh pair.
	if err := userTokenService.ExchangeCode(session, &userToken); err != nil {
		return derp.Wrap(err, location, "Exchanging authorization code")
	}

	// Return the token pair as JSON
	return ctx.JSON(http.StatusOK, userToken.JSONResponse())
}

// postOAuthToken_refresh handles the refresh_token grant: it authenticates the
// client, loads the grant embedded in the refresh token, and rotates it (RFC 6749
// §6) — issuing a new access+refresh pair, or revoking the grant on reuse.
func postOAuthToken_refresh(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthClient model.OAuthClient, transaction model.OAuthUserTokenRequest) error {

	const location = "handler.postOAuthToken_refresh"

	// Parse the refresh token into (grant ID, generation, secret).
	grantID, generation, secret, err := model.ParseRefreshToken(transaction.RefreshToken)

	if err != nil {
		return derp.Wrap(err, location, "Invalid refresh_token")
	}

	// RULE: Authenticate the client. A confidential client must present its
	// client_secret; a public client is identified by client_id, with the refresh
	// token itself as the possession proof (rotation + reuse detection protect it).
	if oauthClient.IsConfidential() {
		if err := oauthClient.ValidateSecret(transaction.ClientSecret); err != nil {
			return derp.Wrap(err, location, "Invalid client_secret", transaction)
		}
	}

	// Load the grant embedded in the refresh token.
	userTokenService := factory.OAuthUserToken()
	userToken := model.NewOAuthUserToken()

	if err := userTokenService.LoadByClientAndID(session, grantID, oauthClient.ClientID, &userToken); err != nil {
		return derp.Wrap(err, location, "Invalid refresh_token")
	}

	// Rotate the grant (or revoke on reuse).
	if err := userTokenService.RefreshGrant(session, &userToken, generation, secret, transaction.Scope); err != nil {
		return derp.Wrap(err, location, "Refreshing token")
	}

	// Return the new token pair as JSON
	return ctx.JSON(http.StatusOK, userToken.JSONResponse())
}

// PostOAuthRevoke handles the OAuth token-revocation endpoint (RFC 7009). It
// deletes the whole grant behind the presented token (access token OR refresh
// token), so both stop working. Per RFC 7009, an unknown/invalid token is still a
// successful (200) revocation.
func PostOAuthRevoke(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostOAuthRevoke"

	// Collect transaction data
	transaction := model.NewOAuthUserTokenRevokeRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Invalid form parameters")
	}

	// Load and authenticate the OAuth Client
	oauthClientService := factory.OAuthClient()
	oauthClient := model.NewOAuthClient()

	if err := oauthClientService.LoadByToken(session, transaction.ClientID, &oauthClient); err != nil {
		return derp.Wrap(err, location, "Invalid client_id")
	}

	// RULE: A confidential client must present its matching client_secret.
	if oauthClient.IsConfidential() {
		if err := oauthClient.ValidateSecret(transaction.ClientSecret); err != nil {
			return derp.Wrap(err, location, "Invalid client_secret")
		}
	}

	// Resolve the grant behind the presented token. Per RFC 7009 an unresolvable
	// token is not an error -- there is simply nothing to revoke.
	grantID, ok := revokeGrantID(factory, transaction.Token)

	if !ok {
		return ctx.JSON(http.StatusOK, map[string]any{})
	}

	// Load the grant (scoped to this client). Already-gone is success.
	userTokenService := factory.OAuthUserToken()
	userToken := model.NewOAuthUserToken()

	if err := userTokenService.LoadByClientAndID(session, grantID, oauthClient.ClientID, &userToken); err != nil {

		if derp.IsNotFound(err) {
			return ctx.JSON(http.StatusOK, map[string]any{})
		}

		return derp.Wrap(err, location, "Loading OAuthUserToken")
	}

	if err := userTokenService.Delete(session, &userToken, "Revoked by Client"); err != nil {
		return derp.Wrap(err, location, "Deleting OAuthUserToken")
	}

	return ctx.JSON(http.StatusOK, map[string]any{})
}

// revokeGrantID resolves the grant ID behind a token presented for revocation. The
// token may be a refresh token (which carries the grant ID directly) or an
// access-token JWT (whose "K" claim holds the grant ID).
func revokeGrantID(factory *service.Factory, token string) (primitive.ObjectID, bool) {

	// A refresh token carries the grant ID as its first segment.
	if grantID, _, _, err := model.ParseRefreshToken(token); err == nil {
		return grantID, true
	}

	// Otherwise treat it as an access-token JWT and read the grant-ID claim.
	parsed, err := factory.JWT().ParseString(token)

	if err != nil {
		return primitive.NilObjectID, false
	}

	if claims, ok := parsed.Claims.(*model.Authorization); ok && !claims.OAuthUserTokenID.IsZero() {
		return claims.OAuthUserTokenID, true
	}

	return primitive.NilObjectID, false
}
