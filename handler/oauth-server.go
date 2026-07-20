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
		"grant_types_supported":                 []string{"authorization_code", "client_credentials", "refresh_token"},
		"code_challenge_methods_supported":      []string{"plain", "S256"}, // PKCE (RFC 7636)
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic"},
		"client_id_metadata_document_supported": true, // https://datatracker.ietf.org/doc/draft-ietf-oauth-client-id-metadata-document/
		"activitypub_object_id_as_client_id":    true, // https://w3id.org/fep/d8c2
		// "scopes_supported":       []string{"read", "write"},
	}

	return ctx.JSON(http.StatusOK, response)
}

func GetOAuthAuthorization(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetOAuthAuthorization"

	// Collect the query parameters
	transaction := model.NewOAuthAuthorizationRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Binding query parameters")
	}

	// Load the OAuth Builder
	builder, err := build.NewOAuthAuthorization(factory, session, transaction, user)

	if err != nil {
		return derp.Wrap(err, location, "Generating Builder")
	}

	// Render the template
	template := factory.Domain().Theme().HTMLTemplate

	if err := template.ExecuteTemplate(ctx.Response(), "oauth", builder); err != nil {
		return derp.Wrap(err, location, "Executing template")
	}

	return nil
}

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

	// If this magic value is passed as the redirect URI, then we just return the token in the <title> tag of the HTML
	// https://docs.joinmastodon.org/methods/apps/#form-data-parameters
	if transaction.RedirectURI == "urn:ietf:wg:oauth:2.0:oob" {
		b := html.New()
		b.HTML()
		b.Head()
		b.Title(userToken.Token)

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

// PostUOAuthToken handles the OAuth token exchange (exchanging code for token)
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

	// Convert transaction.Code => userToken
	userTokenID, err := primitive.ObjectIDFromHex(transaction.Code)

	if err != nil {
		return derp.Wrap(err, location, "Invalid code", transaction)
	}

	// Load the UserToken bound to this code.  This performs NO authentication --
	// the redemption is authenticated below, keyed on the client type.
	userTokenService := factory.OAuthUserToken()
	userToken := model.NewOAuthUserToken()

	if err := userTokenService.LoadByClientAndCode(session, userTokenID, oauthClient.ClientID, &userToken); err != nil {
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

	// Return the Token as JSON
	return ctx.JSON(http.StatusOK, userToken.JSONResponse())
}

func PostOAuthRevoke(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostOAuthRevoke"

	// Collect transaction data
	transaction := model.NewOAuthUserTokenRevokeRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Invalid form parameters")
	}

	// Convert clientID
	clientID, err := primitive.ObjectIDFromHex(transaction.ClientID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid client_id")
	}

	// Load the UserToken
	userTokenService := factory.OAuthUserToken()
	userToken := model.NewOAuthUserToken()

	if err := userTokenService.LoadByClientAndToken(session, clientID, transaction.ClientSecret, transaction.Token, &userToken); err != nil {

		// A token that is already gone needs no revoking
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading OAuthUserToken")
	}

	if err := userTokenService.Delete(session, &userToken, "Revoked by Client"); err != nil {
		return derp.Wrap(err, location, "Deleting OAuthUserToken")
	}

	return ctx.JSON(http.StatusOK, map[string]any{})
}
