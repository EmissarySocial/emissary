package handler

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang-jwt/jwt/v5"
)

func mastodon_PostApplication(serverFactory *server.Factory) func(*http.Request, txn.PostApplication) (object.Application, error) {

	return func(request *http.Request, t txn.PostApplication) (object.Application, error) {

		// Get the domain factory for this request
		factory, err := serverFactory.ByDomainName(request.Host)

		if err != nil {
			return object.Application{}, derp.Wrap(err, "toot.handler.mastodon_PostApplication", "Unrecognized Domain")
		}

		// Collect OAuth Application from the request
		oauthApplication := model.NewOAuthApplication()
		oauthApplication.Name = t.ClientName
		oauthApplication.Website = t.Website
		oauthApplication.RedirectURIs = convert.SliceOfString(t.RedirectURIs)
		oauthApplication.Scopes = strings.Split(t.Scopes, " ")

		// Save the application to the database
		oauthApplicationService := factory.OAuthApplication()
		if err := oauthApplicationService.Save(&oauthApplication, "Created via Mastodon API"); err != nil {
			return object.Application{}, derp.Wrap(err, "toot.handler.mastodon_PostApplication", "Error saving application")
		}

		// Success
		return oauthApplication.ToToot(), nil
	}
}

func mastodon_GetApplication_VerifyCredentials(serverFactory *server.Factory) func(*http.Request, txn.GetApplication_VerifyCredentials) (object.Application, error) {

	const location = "handler.mastodon_GetApplication_VerifyCredentials"

	return func(request *http.Request, t txn.GetApplication_VerifyCredentials) (object.Application, error) {

		// Get the domain factory
		factory, err := serverFactory.ByDomainName(request.Host)

		if err != nil {
			return object.Application{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Validate the JWT token
		jwtService := factory.JWT()
		token, err := jwtService.Parse(request)

		if err != nil {
			return object.Application{}, derp.Wrap(err, location, "Error parsing JWT")
		}

		spew.Dump(token)

		// Get the Application from the database
		oauthApplicationService := factory.OAuthApplication()
		result := model.NewOAuthApplication()
		clientID := token.Claims.(jwt.MapClaims)["client_id"].(string)

		if err := oauthApplicationService.LoadByClientID(clientID, &result); err != nil {
			return object.Application{}, derp.Wrap(err, location, "Error loading application")
		}

		return result.ToToot(), nil
	}
}
