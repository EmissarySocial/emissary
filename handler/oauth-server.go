package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

func GetOAuthAuthorization(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetOAuthAuthorization"

	return func(ctx echo.Context) error {

		// Collect the query parameters
		transaction := model.NewOAuthAuthorizationRequest()

		if err := ctx.Bind(&transaction); err != nil {
			return derp.Wrap(err, location, "Error binding query parameters")
		}

		// Locate the current domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.NewInternalError(location, "Invalid Domain.")
		}

		// Load the OAuth Renderer
		renderer, err := render.NewOAuthAuthorization(factory, transaction)

		if err != nil {
			return derp.Wrap(err, location, "Error Generating Renderer")
		}

		// Render the template
		template := factory.Domain().Theme().HTMLTemplate

		if err := template.ExecuteTemplate(ctx.Response(), "oauth", renderer); err != nil {
			return derp.Wrap(err, location, "Error executing template")
		}

		return nil
	}
}

func PostOAuthAuthorization(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostOAuthAuthorization"

	return func(ctx echo.Context) error {

		// Collect Form parameters
		transaction := model.NewOAuthAuthorizationRequest()

		if err := ctx.Bind(&transaction); err != nil {
			return derp.Wrap(err, location, "Invalid form parameters")
		}

		// Locate the current domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.NewInternalError(location, "Invalid Domain.")
		}

		// Get Authorization
		sterankoContext := ctx.(*steranko.Context)
		authorization := getAuthorization(sterankoContext)

		// Get Application
		applicationService := factory.OAuthApplication()
		application := model.NewOAuthApplication()

		if err := applicationService.LoadByClientID(transaction.ClientID, &application); err != nil {
			return derp.Wrap(err, location, "Error loading OAuth Application")
		}

		// Validate the transaction
		if err := transaction.Validate(application); err != nil {
			return derp.Wrap(err, location, "Invalid transaction")
		}

		// Create a UserToken
		userTokenService := factory.OAuthUserToken()
		userToken, err := userTokenService.Create(application, authorization, transaction)

		if err != nil {
			return derp.Wrap(err, location, "Error creating OAuthUserToken")
		}

		// Complete the transaction based on the grant type
		switch transaction.ResponseType {

		case "code":
			return postOAuthAuthorization_code(ctx, userToken, transaction)

		case "token":
			return postOAuthAuthorization_token(ctx, userTokenService, userToken, transaction)
		}

		return derp.NewBadRequestError(location, "Invalid response type", transaction.ResponseType)
	}
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
func postOAuthAuthorization_token(ctx echo.Context, userTokenService *service.OAuthUserToken, userToken model.OAuthUserToken, transaction model.OAuthAuthorizationRequest) error {

	const location = "handler.postOAuthAuthorization_token"

	// Generate the JWT token for the User
	token, err := userTokenService.JWT(userToken)

	if err != nil {
		return derp.Wrap(err, location, "Error generating JWT")
	}

	// If this magic value is passed as the redirect URI, then we just return the token in the <title> tag of the HTML
	// https://docs.joinmastodon.org/methods/apps/#form-data-parameters
	if transaction.RedirectURI == "urn:ietf:wg:oauth:2.0:oob" {
		b := html.New()
		b.HTML()
		b.Head()
		b.Title(token)

		return ctx.HTML(http.StatusOK, b.String())
	}

	// Otherwise, start building the REAL redirect URI (using the hash fragment)
	redirectURI, err := url.Parse(transaction.RedirectURI)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing redirect_uri")
	}

	redirectURI.Fragment = "access_token=" + token + "&token_type=Bearer"

	// Otherwise, we redirect to the redirect_uri
	return ctx.Redirect(http.StatusFound, redirectURI.String())
}

func PostOAuthToken(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostOAuthToken"

	return func(ctx echo.Context) error {
		return nil
	}
}
