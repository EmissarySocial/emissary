package model

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestOAuthUserTokenRequest_Validate_OmitsSecrets requires that no rejection reports either
// the client's stored secret or the one the request presented
func TestOAuthUserTokenRequest_Validate_OmitsSecrets(t *testing.T) {

	// BUG-173: these errors attached both the client and the request, and each carries a secret.
	// Nothing calls Validate today, so this pins the rule before anything does.
	newClient := func() OAuthClient {
		client := NewOAuthClient()
		client.ClientSecret = "stored-s3cr3t"
		client.RedirectURIs = []string{"https://app.example.com/callback"}
		return client
	}

	// requireRejected fails unless the request is rejected with the message, carrying no secret
	requireRejected := func(t *testing.T, client OAuthClient, request OAuthUserTokenRequest, message string) {
		t.Helper()

		err := request.Validate(client)
		require.Equal(t, message, derp.Message(err))
		secretcheck.RequireAbsent(t, err, "stored-s3cr3t")
		secretcheck.RequireAbsent(t, err, "presented-s3cr3t")
	}

	t.Run("WrongClientID", func(t *testing.T) {
		client := newClient()
		requireRejected(t, client, OAuthUserTokenRequest{ClientID: "someone-else", ClientSecret: "presented-s3cr3t"}, "Invalid client_id")
	})

	t.Run("WrongClientSecret", func(t *testing.T) {
		client := newClient()
		requireRejected(t, client, OAuthUserTokenRequest{ClientID: client.ClientID.Hex(), ClientSecret: "presented-s3cr3t"}, "Invalid client_secret")
	})

	t.Run("WrongRedirectURI", func(t *testing.T) {
		client := newClient()
		client.ClientSecret = "presented-s3cr3t"
		requireRejected(t, client, OAuthUserTokenRequest{ClientID: client.ClientID.Hex(), ClientSecret: "presented-s3cr3t", RedirectURI: "https://evil.example.com"}, "Invalid redirect_uri")
	})
}
