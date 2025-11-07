package mastodon

import (
	"strings"
	"time"

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
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Application{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Application{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Collect OAuth Application from the request
		oauthClient := model.NewOAuthClient()
		oauthClient.Name = t.ClientName
		oauthClient.Website = t.Website
		oauthClient.RedirectURIs = convert.SliceOfString(t.RedirectURIs)
		oauthClient.Scopes = strings.Split(t.Scopes, " ")

		// Save the application to the database
		oauthClientService := factory.OAuthClient()
		if err := oauthClientService.Save(session, &oauthClient, "Created via Mastodon API"); err != nil {
			return object.Application{}, derp.Wrap(err, location, "Unable to save application")
		}

		// Success
		return oauthClient.Toot(), nil
	}
}

func GetApplication_VerifyCredentials(serverFactory *server.Factory) func(model.Authorization, txn.GetApplication_VerifyCredentials) (object.Application, error) {

	const location = "handler.mastodon_GetApplication_VerifyCredentials"

	return func(auth model.Authorization, t txn.GetApplication_VerifyCredentials) (object.Application, error) {

		// Get the domain factory
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Application{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Application{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()
		// Get the Application from the database
		oauthClientService := factory.OAuthClient()
		result := model.NewOAuthClient()

		// Try to load the client record from the database
		if err := oauthClientService.LoadByClientID(session, auth.ClientID, &result); err != nil {
			return object.Application{}, derp.Wrap(err, location, "Unable to load application")
		}

		return result.Toot(), nil
	}
}
