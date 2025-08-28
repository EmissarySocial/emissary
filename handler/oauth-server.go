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

func GetOAuthAuthorization(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetOAuthAuthorization"

	// Collect the query parameters
	transaction := model.NewOAuthAuthorizationRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Error binding query parameters")
	}

	// Load the OAuth Builder
	builder, err := build.NewOAuthAuthorization(factory, session, transaction)

	if err != nil {
		return derp.Wrap(err, location, "Error Generating Builder")
	}

	// Render the template
	template := factory.Domain().Theme().HTMLTemplate

	if err := template.ExecuteTemplate(ctx.Response(), "oauth", builder); err != nil {
		return derp.Wrap(err, location, "Error executing template")
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
		return derp.Wrap(err, location, "Error loading OAuth Application")
	}

	// Validate the transaction
	if err := transaction.Validate(application); err != nil {
		return derp.Wrap(err, location, "Invalid transaction")
	}

	// Create a UserToken
	userTokenService := factory.OAuthUserToken()
	userToken, err := userTokenService.Create(session, application, authorization, transaction)

	if err != nil {
		return derp.Wrap(err, location, "Error creating OAuthUserToken")
	}

	// Complete the transaction based on the grant type
	switch transaction.ResponseType {

	case "code":
		return postOAuthAuthorization_code(ctx, userToken, transaction)

	case "token":
		return postOAuthAuthorization_token(ctx, userToken, transaction)
	}

	return derp.BadRequestError(location, "Invalid response type", transaction.ResponseType)
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
		return derp.Wrap(err, location, "Error parsing redirect_uri")
	}

	redirectURI.Fragment = "access_token=" + userToken.Token + "&token_type=Bearer"

	// Otherwise, we redirect to the redirect_uri
	return ctx.Redirect(http.StatusFound, redirectURI.String())
}

func PostOAuthToken(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostOAuthToken"

	// Collect transaction data
	transaction := model.NewOAuthUserTokenRequest()

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Invalid form parameters")
	}

	// Convert client ID
	clientID, err := primitive.ObjectIDFromHex(transaction.ClientID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid client_id")
	}

	// Convert transaction.Code => userToken
	userTokenID, err := primitive.ObjectIDFromHex(transaction.Code)

	if err != nil {
		return derp.Wrap(err, location, "Invalid code")
	}

	// Load the UserToken
	userTokenService := factory.OAuthUserToken()
	userToken := model.NewOAuthUserToken()

	if err := userTokenService.LoadByClientAndCode(session, userTokenID, clientID, transaction.ClientSecret, &userToken); err != nil {
		return derp.Wrap(err, location, "Error loading OAuthUserToken")
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

	err = userTokenService.LoadByClientAndToken(session, clientID, transaction.ClientSecret, transaction.Token, &userToken)

	if derp.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return derp.Wrap(err, location, "Error loading OAuthUserToken")
	}

	if err := userTokenService.Delete(session, &userToken, "Revoked by Client"); err != nil {
		return derp.Wrap(err, location, "Error deleting OAuthUserToken")
	}

	return ctx.JSON(http.StatusOK, map[string]any{})
}
