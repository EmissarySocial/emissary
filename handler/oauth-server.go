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
	"github.com/benpate/toot/scope"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetOAuthWellKnown serves the OAuth authorization-server metadata document (RFC 8414)
func GetOAuthWellKnown(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	return ctx.JSON(http.StatusOK, oauthMetadata(factory.Host()))
}

// oauthMetadata returns the authorization-server metadata document (RFC 8414) for a host, which is
// scheme, hostname, and port with no trailing slash
func oauthMetadata(host string) map[string]any {

	// Both endpoints read client_secret from the form body only, and a public client sends none
	clientAuthMethods := []string{"client_secret_post", "none"}

	return map[string]any{
		"issuer":                                         host,
		"authorization_endpoint":                         host + "/oauth/authorize",
		"token_endpoint":                                 host + "/oauth/token",
		"revocation_endpoint":                            host + "/oauth/revoke",
		"response_types_supported":                       []string{"code", "token"},
		"response_modes_supported":                       []string{"query", "fragment"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"scopes_supported":                               oauthScopesSupported(),
		"code_challenge_methods_supported":               []string{"plain", "S256"}, // PKCE (RFC 7636)
		"token_endpoint_auth_methods_supported":          clientAuthMethods,
		"revocation_endpoint_auth_methods_supported":     clientAuthMethods,
		"authorization_response_iss_parameter_supported": true, // RFC 9207
		"client_id_metadata_document_supported":          true, // https://datatracker.ietf.org/doc/draft-ietf-oauth-client-id-metadata-document/
		"activitypub_object_id_as_client_id":             true, // https://w3id.org/fep/d8c2
	}
}

// oauthScopesSupported returns the scopes the metadata document advertises: the families that grant
// something here, and none of the narrower scopes inside them
func oauthScopesSupported() []string {

	// RFC 8414 lets a server leave supported scopes out.  Moderation and `follow` grant nothing here
	return []string{
		model.OAuthUserTokenScopeProfile,
		model.OAuthUserTokenScopeEmail,
		scope.Read,
		scope.Write,
		scope.Push,
		model.OAuthUserTokenScopeActivityPubPortability,
	}
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

// RouteOAuthAuthorization sends a POST to the authorization endpoint that redeems a code to redemption, and
// every other POST, which is the consent form's submit, to consent
func RouteOAuthAuthorization(redemption echo.HandlerFunc, consent echo.HandlerFunc) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// A redemption comes from another server with no session, so it must never meet consent's sign-in check
		if isCodeRedemption(ctx.Request()) {
			return redemption(ctx)
		}

		return consent(ctx)
	}
}

// isCodeRedemption returns TRUE if a POST body redeems an authorization code.  The consent form posts neither
// `code` nor `grant_type`, so either one marks a redemption (IndieAuth §5.3.1).
func isCodeRedemption(request *http.Request) bool {

	// A body that cannot be parsed is not a redemption.  ParseForm keeps the fields it could read, so the
	// consent handler binds those and validates them as usual.
	if err := request.ParseForm(); err != nil {
		return false
	}

	// Only the body counts, so a query string cannot move a consent submit to the other branch
	if request.PostForm.Get("grant_type") != "" {
		return true
	}

	return request.PostForm.Get("code") != ""
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

	// RULE: A User who refuses is sent back to the client with the refusal, and no grant is written
	if transaction.IsDenied() {
		return postOAuthAuthorization_deny(ctx, factory.Host(), transaction)
	}

	// Create a UserToken
	userTokenService := factory.OAuthUserToken()
	userToken, err := userTokenService.Create(session, application, authorization, transaction)

	if err != nil {
		return derp.Wrap(err, location, "Creating OAuthUserToken")
	}

	// Complete the transaction based on the grant type.  The issuer is the one the metadata
	// publishes, because clients compare the two as strings (RFC 9207)
	switch transaction.ResponseType {

	case "code":
		return postOAuthAuthorization_code(ctx, factory.Host(), userToken, transaction)

	case "token":
		return postOAuthAuthorization_token(ctx, factory.Host(), userToken, transaction)
	}

	return derp.BadRequest(location, "Invalid response type", transaction.ResponseType)
}

// postOAuthAuthorization_code handles `code` grant types used by server authentication flow
func postOAuthAuthorization_code(ctx echo.Context, issuer string, userToken model.OAuthUserToken, transaction model.OAuthAuthorizationRequest) error {

	// If this magic value is passed as the redirect URI, then we return the
	// authorization code in the <title> tag of the HTML for the user to copy into
	// their client, which then exchanges it at /oauth/token.
	// https://docs.joinmastodon.org/methods/apps/#form-data-parameters
	if transaction.RedirectURI == model.OAuthRedirectURIOutOfBand {
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

	// Add the code and the issuer to the URI (RFC 9207, IndieAuth §5.2.1)
	queryString := redirectURI.Query()
	queryString.Set("code", userToken.Code())
	queryString.Set("state", transaction.State)
	queryString.Set("iss", issuer)
	redirectURI.RawQuery = queryString.Encode()

	return ctx.Redirect(http.StatusFound, redirectURI.String())
}

// postOAuthAuthorization_token handles `token` grant types used by the client-side authentication flow
func postOAuthAuthorization_token(ctx echo.Context, issuer string, userToken model.OAuthUserToken, transaction model.OAuthAuthorizationRequest) error {

	const location = "handler.postOAuthAuthorization_token"

	// If this magic value is passed as the redirect URI, then we just return the token in the <title> tag of the HTML
	// https://docs.joinmastodon.org/methods/apps/#form-data-parameters
	if transaction.RedirectURI == model.OAuthRedirectURIOutOfBand {
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

	// RULE: The fragment is form-encoded (RFC 6749 §4.2.2), so a state cannot add parameters of its own
	fragment := url.Values{}
	fragment.Set("access_token", userToken.Token)
	fragment.Set("state", transaction.State)
	fragment.Set("token_type", "Bearer")
	fragment.Set("iss", issuer)

	// The encoded fragment is appended, because url.URL would escape its "%" a second time
	redirectURI.Fragment = ""
	redirectURI.RawFragment = ""

	return ctx.Redirect(http.StatusFound, redirectURI.String()+"#"+fragment.Encode())
}

// postOAuthAuthorization_deny sends the User back to the client with `error=access_denied` (RFC 6749 §4.1.2.1)
func postOAuthAuthorization_deny(ctx echo.Context, issuer string, transaction model.OAuthAuthorizationRequest) error {

	// RULE: An out-of-band request has no address to answer, so the User goes home
	if transaction.RedirectURI == model.OAuthRedirectURIOutOfBand {
		return ctx.Redirect(http.StatusFound, "/")
	}

	return ctx.Redirect(http.StatusFound, oauthDenialURL(transaction, issuer))
}

// oauthDenialURL returns a validated request's redirect_uri carrying the refusal, in the query for a
// code request and in the fragment for a token request
func oauthDenialURL(transaction model.OAuthAuthorizationRequest, issuer string) string {

	answer := url.Values{}
	answer.Set("error", "access_denied")
	answer.Set("state", transaction.State)
	answer.Set("iss", issuer)

	// A redirect_uri that will not parse was refused by Validate, so this only guards the call below
	redirectURI, err := url.Parse(transaction.RedirectURI)

	if err != nil {
		return "/"
	}

	// RULE: An implicit request is answered in the fragment, encoded on its own so a state adds no parameters
	if transaction.ResponseType == "token" {
		redirectURI.Fragment = ""
		redirectURI.RawFragment = ""
		return redirectURI.String() + "#" + answer.Encode()
	}

	// Every other request is answered in the query, keeping whatever the redirect_uri already carries
	queryString := redirectURI.Query()

	for name := range answer {
		queryString.Set(name, answer.Get(name))
	}

	redirectURI.RawQuery = queryString.Encode()
	return redirectURI.String()
}

// PostOAuthAuthorizationRedemption redeems an authorization code at the authorization endpoint, for a client that
// needs only the User's identity, and answers with `me` and never a token (IndieAuth §5.3.2)
func PostOAuthAuthorizationRedemption(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostOAuthAuthorizationRedemption"

	// Collect transaction data
	transaction := model.NewOAuthUserTokenRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return oauthTokenError(ctx, "invalid_request", derp.Wrap(err, location, "Invalid form parameters", derp.WithBadRequest()))
	}

	// Load the OAuth Client (from ID or from ActorID).  Errors name the client_id, never the transaction.
	oauthClient := model.NewOAuthClient()

	if err := factory.OAuthClient().LoadByToken(session, transaction.ClientID, &oauthClient); err != nil {
		return oauthTokenError(ctx, "invalid_client", derp.Wrap(err, location, "Invalid client_id", transaction.ClientID))
	}

	// RULE: grant_type is required (IndieAuth §5.3.1), so a request that names none redeems nothing
	if transaction.GrantType == "" {
		return oauthTokenError(ctx, "invalid_request", derp.BadRequest(location, "Missing grant_type"))
	}

	// RULE: Only an authorization code is redeemed here
	if transaction.GrantType != "authorization_code" {
		return oauthTokenError(ctx, "unsupported_grant_type", derp.BadRequest(location, "Unsupported grant_type", transaction.GrantType))
	}

	// Load the grant behind the code, and the User it was issued for.  The helper's message is the
	// client's error_description, so it is not wrapped again.
	userToken := model.NewOAuthUserToken()
	user := model.NewUser()

	if errorCode, err := loadAuthorizationCode(factory, session, oauthClient, transaction, &userToken, &user); err != nil {
		return oauthTokenError(ctx, errorCode, err)
	}

	// RULE: This endpoint answers only with an identity, so a code whose User has none is refused unspent
	if user.IndieAuthMe() == "" {
		return oauthTokenError(ctx, "invalid_grant", derp.BadRequest(location, "This server vouches for no identity to return", transaction.ClientID))
	}

	// Redeem the single-use code, minting nothing, whatever scope it carries
	if err := factory.OAuthUserToken().RedeemCode(session, &userToken); err != nil {
		return oauthTokenError(ctx, "invalid_grant", derp.Wrap(err, location, "Redeeming authorization code"))
	}

	// Return the User's identity, and the profile that the grant's scopes release
	return ctx.JSON(http.StatusOK, userToken.IdentityResponse(user))
}

// PostOAuthToken handles the OAuth token endpoint (RFC 6749 §3.2), exchanging an authorization
// code, or rotating a refresh token, for a new access and refresh pair
func PostOAuthToken(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostOAuthToken"

	// Collect transaction data
	transaction := model.NewOAuthUserTokenRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return oauthTokenError(ctx, "invalid_request", derp.Wrap(err, location, "Invalid form parameters", derp.WithBadRequest()))
	}

	// Load the OAuth Client (from ID or from ActorID)
	oauthClientService := factory.OAuthClient()
	oauthClient := model.NewOAuthClient()

	// Errors name the client_id, never the transaction, which carries the client_secret, the
	// code, the refresh_token, and the code_verifier.  Reported errors are stored verbatim.
	if err := oauthClientService.LoadByToken(session, transaction.ClientID, &oauthClient); err != nil {
		return oauthTokenError(ctx, "invalid_client", derp.Wrap(err, location, "Invalid client_id", transaction.ClientID))
	}

	// RULE: grant_type is required (RFC 6749 §4.1.3 and §6)
	if transaction.GrantType == "" {
		return oauthTokenError(ctx, "invalid_request", derp.BadRequest(location, "Missing grant_type"))
	}

	// Dispatch on grant_type
	switch transaction.GrantType {

	case "authorization_code":
		return postOAuthToken_authorizationCode(ctx, factory, session, oauthClient, transaction)

	case "refresh_token":
		return postOAuthToken_refresh(ctx, factory, session, oauthClient, transaction)
	}

	return oauthTokenError(ctx, "unsupported_grant_type", derp.BadRequest(location, "Unsupported grant_type", transaction.GrantType))
}

// postOAuthToken_authorizationCode exchanges a single-use authorization code for an access and
// refresh pair, once the client proves itself by secret or by PKCE
func postOAuthToken_authorizationCode(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthClient model.OAuthClient, transaction model.OAuthUserTokenRequest) error {

	const location = "handler.postOAuthToken_authorizationCode"

	// Load the grant behind the code, and the User it was issued for.  The helper's message is the
	// client's error_description, so it is not wrapped again.
	userToken := model.NewOAuthUserToken()
	user := model.NewUser()

	if errorCode, err := loadAuthorizationCode(factory, session, oauthClient, transaction, &userToken, &user); err != nil {
		return oauthTokenError(ctx, errorCode, err)
	}

	// RULE: A code with no scope gets no token, so it is refused unspent when there is no identity to return either
	if !userToken.IsRedeemableFor(user) {
		return oauthTokenError(ctx, "invalid_grant", derp.BadRequest(location, "This code carries no scope, and this server vouches for no identity to return", transaction.ClientID))
	}

	// Redeem the single-use code, minting an access+refresh pair when the grant carries a scope
	if err := factory.OAuthUserToken().ExchangeCode(session, &userToken); err != nil {
		return oauthTokenError(ctx, "invalid_grant", derp.Wrap(err, location, "Exchanging authorization code"))
	}

	// Return the tokens, and the User's identity when the domain vouches for them
	return ctx.JSON(http.StatusOK, userToken.TokenResponse(user))
}

// postOAuthToken_refresh rotates a refresh token into a new access and refresh pair (RFC 6749
// §6), revoking the whole grant when a token is reused
func postOAuthToken_refresh(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthClient model.OAuthClient, transaction model.OAuthUserTokenRequest) error {

	const location = "handler.postOAuthToken_refresh"

	// Parse the refresh token into (grant ID, generation, secret).
	grantID, generation, secret, err := model.ParseRefreshToken(transaction.RefreshToken)

	if err != nil {
		return oauthTokenError(ctx, "invalid_grant", derp.Wrap(err, location, "Invalid refresh_token"))
	}

	// RULE: Authenticate the client. A confidential client must present its
	// client_secret; a public client is identified by client_id, with the refresh
	// token itself as the possession proof (rotation + reuse detection protect it).
	if oauthClient.IsConfidential() {
		if err := oauthClient.ValidateSecret(transaction.ClientSecret); err != nil {
			return oauthTokenError(ctx, "invalid_client", derp.Wrap(err, location, "Invalid client_secret", transaction.ClientID))
		}
	}

	// Load the grant embedded in the refresh token.
	userTokenService := factory.OAuthUserToken()
	userToken := model.NewOAuthUserToken()

	if err := userTokenService.LoadByClientAndID(session, grantID, oauthClient.ClientID, &userToken); err != nil {
		return oauthTokenError(ctx, "invalid_grant", derp.Wrap(err, location, "Invalid refresh_token"))
	}

	// Rotate the grant, or revoke it on reuse.  A refusal commits, so a revocation sticks.
	if err := userTokenService.RefreshGrant(session, &userToken, generation, secret, transaction.Scope); err != nil {
		return oauthTokenError(ctx, "invalid_grant", derp.Wrap(err, location, "Refreshing token"))
	}

	// Return the new token pair as JSON
	return ctx.JSON(http.StatusOK, userToken.JSONResponse())
}

// loadAuthorizationCode loads the grant behind a code and the User it was issued for, once the client has proven
// itself and the request matches the authorization request.  A refusal comes with its RFC 6749 §5.2 error code.
func loadAuthorizationCode(factory *service.Factory, session data.Session, oauthClient model.OAuthClient, transaction model.OAuthUserTokenRequest, userToken *model.OAuthUserToken, user *model.User) (string, error) {

	const location = "handler.loadAuthorizationCode"

	// Convert transaction.Code => grant ID
	userTokenID, err := primitive.ObjectIDFromHex(transaction.Code)

	if err != nil {
		return "invalid_grant", derp.Wrap(err, location, "Invalid code", transaction.ClientID, derp.WithBadRequest())
	}

	// Load the grant bound to this code. This performs NO authentication -- the
	// redemption is authenticated below, keyed on the client type.
	if err := factory.OAuthUserToken().LoadByClientAndID(session, userTokenID, oauthClient.ClientID, userToken); err != nil {
		return "invalid_grant", derp.Wrap(err, location, "Loading OAuthUserToken", transaction.ClientID)
	}

	// RULE: Authenticate the code redemption (RFC 8252 / OAuth 2.1).  A
	// confidential client must present its matching client_secret; a public client
	// (no secret -- e.g. a CIMD/FEP-d8c2 client) must instead have bound its code
	// to a PKCE challenge, so an intercepted code is useless without the verifier.
	if oauthClient.IsConfidential() {
		if err := oauthClient.ValidateSecret(transaction.ClientSecret); err != nil {
			return "invalid_client", derp.Wrap(err, location, "Invalid client_secret", transaction.ClientID)
		}
	} else if !userToken.HasPKCEChallenge() {
		return "invalid_grant", derp.BadRequest(location, "This client must use PKCE (a code_verifier is required)", transaction.ClientID)
	}

	// RULE: PKCE (RFC 7636). If the code was issued with a code_challenge, a
	// matching code_verifier is required to redeem it.
	if err := userToken.VerifyPKCE(transaction.CodeVerifier); err != nil {
		return "invalid_grant", derp.Wrap(err, location, "Invalid PKCE code_verifier")
	}

	// RULE: The redirect_uri is the one the code was issued to (RFC 6749 §4.1.3, IndieAuth §5.3.1)
	if err := userToken.VerifyRedirectURI(oauthClient, transaction.RedirectURI); err != nil {
		return "invalid_grant", derp.Wrap(err, location, "Invalid redirect_uri", transaction.ClientID)
	}

	// Load the User the code was issued for
	if err := factory.User().LoadByID(session, userToken.UserID, user); err != nil {
		return "invalid_grant", derp.Wrap(err, location, "Loading User", transaction.ClientID)
	}

	// Ready to redeem
	return "", nil
}

// oauthTokenError answers a token request the client got wrong as RFC 6749 §5.2 JSON, and
// returns any other failure for errorHandler to report
func oauthTokenError(ctx echo.Context, errorCode string, err error) error {

	// RULE: A server fault is returned, so it is reported and its transaction rolls back
	if !derp.IsClientError(err) {
		return err
	}

	// A client's mistake is answered here, so it is never reported, and the transaction commits
	return ctx.JSON(http.StatusBadRequest, map[string]any{
		"error":             errorCode,
		"error_description": derp.Message(err),
	})
}

// PostOAuthRevoke handles the token-revocation endpoint (RFC 7009) by deleting the whole
// grant behind the presented access or refresh token
func PostOAuthRevoke(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostOAuthRevoke"

	// Deleting the grant stops both of its tokens.  Per RFC 7009, an unknown or invalid token
	// is still a successful (200) revocation.

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

// revokeGrantID returns the grant ID behind a refresh token or an access-token JWT that was
// presented for revocation
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
