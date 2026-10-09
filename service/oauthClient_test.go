package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// testClientSecret stands in for an OAuth client secret that must never reach an error
const testClientSecret = "oauth-client-secret-s3cr3t"

// TestOAuthClient_OmitsSecretFromErrors requires that no failure reports the client's secret
func TestOAuthClient_OmitsSecretFromErrors(t *testing.T) {

	// BUG-173: ClientSecret is json:"-" but bson:"clientSecret", so the console hid it and
	// the error log stored it.
	service := OAuthClient{oauthUserTokenService: &OAuthUserToken{}}

	// requireSite fails unless err came from the named site, carrying none of the client's secret.
	// Save generates a new secret for a new client, so the secret is read after the call.
	requireSite := func(t *testing.T, err error, client model.OAuthClient, location string, message string) {
		t.Helper()
		require.Equal(t, location, derp.Location(err))
		require.Equal(t, message, derp.Message(err))
		require.NotEmpty(t, client.ClientSecret)
		secretcheck.RequireAbsent(t, err, client.ClientSecret)
	}

	t.Run("Invalid", func(t *testing.T) {
		client := newTestOAuthClient()
		client.Name = ""
		err := service.Save(brokenSession{}, &client, "test")

		// The whole chain is checked, including rosetta's schema error beneath this layer
		requireSite(t, err, client, "service.OAuthClient.Save", "Validating OAuthClient using OAuthClientSchema")
	})

	t.Run("Save", func(t *testing.T) {
		client := newTestOAuthClient()
		err := service.Save(brokenSession{}, &client, "test")
		requireSite(t, err, client, "service.OAuthClient.Save", "Saving OAuthClient")
	})

	t.Run("Delete", func(t *testing.T) {
		client := newTestOAuthClient()
		err := service.Delete(brokenSession{}, &client, "test")
		requireSite(t, err, client, "service.OAuthClient.Delete", "Deleting OAuthClient")
	})

	t.Run("DeleteGrants", func(t *testing.T) {
		client := newTestOAuthClient()
		grant := model.NewOAuthUserToken()
		grant.ClientID = client.ClientID

		session := brokenSession{
			records:  map[string][]data.Object{"OAuthUserToken": {&grant}},
			writable: map[string]bool{"OAuthClient": true},
		}

		err := service.Delete(session, &client, "test")
		requireSite(t, err, client, "service.OAuthClient.Delete", "Deleting attachments")
	})
}

// newTestOAuthClient returns a valid client carrying testClientSecret
func newTestOAuthClient() model.OAuthClient {
	client := model.NewOAuthClient()
	client.Name = "Test Client"
	client.ClientSecret = testClientSecret
	return client
}
