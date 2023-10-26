package mastodon

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

func PostApplication(serverFactory *server.Factory) func(model.Authorization, txn.PostApplication) (object.Application, error) {

	const location = "handler.mastodon_PostApplication"

	return func(authorization model.Authorization, t txn.PostApplication) (object.Application, error) {

		// Get the domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Application{}, derp.Wrap(err, location, "Unrecognized Domain")
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
			return object.Application{}, derp.Wrap(err, location, "Error saving application")
		}

		// Success
		return oauthClient.Toot(), nil
	}
}

func GetApplication_VerifyCredentials(serverFactory *server.Factory) func(model.Authorization, txn.GetApplication_VerifyCredentials) (object.Application, error) {

	const location = "handler.mastodon_GetApplication_VerifyCredentials"

	return func(auth model.Authorization, t txn.GetApplication_VerifyCredentials) (object.Application, error) {

		// Get the domain factory
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Application{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get the Application from the database
		oauthClientService := factory.OAuthClient()
		result := model.NewOAuthClient()

		// Try to load the client record from the database
		if err := oauthClientService.LoadByClientID(auth.ClientID, &result); err != nil {
			return object.Application{}, derp.Wrap(err, location, "Error loading application")
		}

		return result.Toot(), nil
	}
}
