package service

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/cimd"
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

// TestOAuthClient_LoadByToken_RejectsEmpty requires an empty client_id to fail before any lookup, never matching an arbitrary client
func TestOAuthClient_LoadByToken_RejectsEmpty(t *testing.T) {

	service := OAuthClient{}
	client := model.NewOAuthClient()

	// brokenSession fails any database call, so reaching the database would report a different site
	err := service.LoadByToken(brokenSession{}, "", &client)
	require.Error(t, err)
	require.Equal(t, "service.OAuthClient.LoadByToken", derp.Location(err))
}

// TestOAuthClient_LoadByToken_RejectsInvalidURL requires a client_id that is neither an ObjectID nor a valid
// URL client_id to be refused as a bad request before any lookup
func TestOAuthClient_LoadByToken_RejectsInvalidURL(t *testing.T) {

	service := OAuthClient{}

	for _, token := range []string{"not-a-url", "https://127.0.0.1/", "https://app.example.com/#x"} {
		client := model.NewOAuthClient()
		err := service.LoadByToken(brokenSession{}, token, &client)
		require.True(t, derp.IsBadRequest(err), "%q answers %d", token, derp.ErrorCode(err))
	}
}

// TestNewURLClient verifies the client built for a URL client_id: named by its hostname when it publishes
// nothing, and from its metadata otherwise, keeping only values the schema accepts
func TestNewURLClient(t *testing.T) {

	schema := (&OAuthClient{}).Schema()

	valid := func(client model.OAuthClient) {
		t.Helper()
		_, err := schema.Validate(&client)
		require.NoError(t, err)
	}

	t.Run("no metadata", func(t *testing.T) {
		client := newURLClient("https://fed.brid.gy/", cimd.Metadata{ClientName: "Ignored"}, false)
		require.Equal(t, "https://fed.brid.gy/", client.ClientURL)
		require.Equal(t, "fed.brid.gy", client.Name)
		require.Empty(t, client.RedirectURIs)
		require.Empty(t, client.IconURL)
		require.Empty(t, client.Website)
		require.False(t, client.IsConfidential())
		valid(client)
	})

	t.Run("metadata", func(t *testing.T) {
		client := newURLClient("https://app.example.com/client.json", cimd.Metadata{
			ClientID:     "https://app.example.com/client.json",
			ClientName:   "  Example App  ",
			LogoURI:      "https://app.example.com/logo.png",
			ClientURI:    "https://app.example.com/",
			RedirectURIs: []string{"https://app.example.com/callback"},
		}, true)

		require.Equal(t, "Example App", client.Name)
		require.Equal(t, "https://app.example.com/logo.png", client.IconURL)
		require.Equal(t, "https://app.example.com/", client.Website, "a full client_uri, which the schema's url format accepts")
		require.Equal(t, []string{"https://app.example.com/callback"}, []string(client.RedirectURIs))
		valid(client)
	})

	t.Run("malformed metadata is dropped, not refused", func(t *testing.T) {
		client := newURLClient("https://app.example.com/client.json", cimd.Metadata{
			ClientName: strings.Repeat("é", 200),
			LogoURI:    "javascript:alert(1)",
			ClientURI:  "app.example.com",
		}, true)

		require.Equal(t, strings.Repeat("é", model.OAuthClientNameMaxLength), client.Name)
		require.Empty(t, client.IconURL)
		require.Empty(t, client.Website)
		valid(client)

		blank := newURLClient("https://app.example.com/client.json", cimd.Metadata{ClientName: "   "}, true)
		require.Equal(t, "app.example.com", blank.Name)
	})
}
