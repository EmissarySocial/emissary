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
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func mastodon_PostApplication(serverFactory *server.Factory) func(*http.Request, txn.PostApplication) (object.Application, error) {

	return func(request *http.Request, t txn.PostApplication) (object.Application, error) {

		// Get the domain factory for this request
		factory, err := serverFactory.ByDomainName(request.Host)

		if err != nil {
			return object.Application{}, derp.Wrap(err, "toot.handler.mastodon_PostApplication", "Unrecognized Domain")
		}

		// Collect OAuth Application from the request
		oauthClient := model.NewOAuthClient()
		oauthClient.Name = t.ClientName
		oauthClient.Website = t.Website
		oauthClient.RedirectURIs = convert.SliceOfString(t.RedirectURIs)
		oauthClient.Scopes = strings.Split(t.Scopes, " ")

		// Save the application to the database
		oauthClientService := factory.OAuthClient()
		if err := oauthClientService.Save(&oauthClient, "Created via Mastodon API"); err != nil {
			return object.Application{}, derp.Wrap(err, "toot.handler.mastodon_PostApplication", "Error saving application")
		}

		// Success
		return oauthClient.ToToot(), nil
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

		// Get the Application from the database
		oauthClientService := factory.OAuthClient()
		result := model.NewOAuthClient()
		clientString := token.Claims.(jwt.MapClaims)["client_id"].(string)

		// Convert the clientID
		clientID, err := primitive.ObjectIDFromHex(clientString)

		if err != nil {
			return object.Application{}, derp.Wrap(err, location, "Invalid client_id")
		}

		// Try to load the client record from the database
		if err := oauthClientService.LoadByClientID(clientID, &result); err != nil {
			return object.Application{}, derp.Wrap(err, location, "Error loading application")
		}

		return result.ToToot(), nil
	}
}
