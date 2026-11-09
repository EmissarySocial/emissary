package model

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestOAuthAuthorizationRequest_RedirectURI verifies how Validate fills in and checks the redirect_uri,
// with every refusal a bad request
func TestOAuthAuthorizationRequest_RedirectURI(t *testing.T) {

	// Every request carries a PKCE challenge, so only the redirect_uri decides the outcome
	validate := func(client OAuthClient, redirectURI string) (OAuthAuthorizationRequest, error) {
		request := OAuthAuthorizationRequest{ResponseType: "code", RedirectURI: redirectURI, CodeChallenge: "challenge"}
		err := request.Validate(client)
		return request, err
	}

	refused := func(err error, reason string) {
		t.Helper()
		require.Error(t, err, reason)
		require.True(t, derp.IsBadRequest(err), "%s answers %d", reason, derp.ErrorCode(err))
	}

	registered := NewOAuthClient()
	registered.RedirectURIs = append(registered.RedirectURIs, "https://app.example.com/first", "https://app.example.com/second")

	empty := NewOAuthClient()

	urlClient := NewOAuthClient()
	urlClient.ClientURL = "https://fed.brid.gy/"

	// A registered client defaults to its first redirect_uri, and accepts any it lists
	request, err := validate(registered, "")
	require.NoError(t, err)
	require.Equal(t, "https://app.example.com/first", request.RedirectURI)

	_, err = validate(registered, "https://app.example.com/second")
	require.NoError(t, err)

	_, err = validate(registered, "https://app.example.com/third")
	refused(err, "an unlisted redirect_uri")

	// A client that lists nothing is a bad request, not a server error
	_, err = validate(empty, "")
	refused(err, "no redirect_uri, and none listed")

	_, err = validate(empty, "https://app.example.com/callback")
	refused(err, "a redirect_uri for a client that lists none")

	// A URL client must send one, on its own origin
	_, err = validate(urlClient, "")
	refused(err, "a URL client with no redirect_uri")

	request, err = validate(urlClient, "https://fed.brid.gy/oauth/indieauth/finish")
	require.NoError(t, err)
	require.Equal(t, "https://fed.brid.gy/oauth/indieauth/finish", request.RedirectURI)

	_, err = validate(urlClient, "https://evil.example/finish")
	refused(err, "a URL client's redirect_uri on another origin")
}
