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

// FuzzOAuthAuthorizationRequest_Validate requires every request that Validate accepts to be one the server can honor
func FuzzOAuthAuthorizationRequest_Validate(f *testing.F) {

	f.Add("https://client.example.com/", "", "https://client.example.com/callback", "", "code", "challenge", "S256", "", "")
	f.Add("https://client.example.com/", "", "https://evil.example.com/callback", "profile", "code", "challenge", "", "", "")
	f.Add("", "https://registered.example.com/callback", "", "read write", "token", "", "", "secret", "read")
	f.Add("", "https://registered.example.com/callback", "https://evil.example.com/", "admin:write", "code", "x", "S512", "", "read")
	f.Add("https://client.example.com/", "", "https://client.example.com/callback", "profile", "", "", "", "", "")
	f.Add("", "https://registered.example.com/callback", "", "", "code", "", "", "", "")

	f.Fuzz(func(t *testing.T, clientURL, listed, redirectURI, scope, responseType, challenge, method, secret, clientScope string) {

		client := NewOAuthClient()
		client.ClientURL = clientURL
		client.ClientSecret = secret

		if listed != "" {
			client.RedirectURIs = append(client.RedirectURIs, listed)
		}

		if clientScope != "" {
			client.Scopes = append(client.Scopes, clientScope)
		}

		request := OAuthAuthorizationRequest{
			ResponseType:        responseType,
			RedirectURI:         redirectURI,
			Scope:               scope,
			CodeChallenge:       challenge,
			CodeChallengeMethod: method,
		}

		if err := request.Validate(client); err != nil {
			return
		}

		// RULE: An accepted request sends the User only where the client may send them
		require.True(t, client.AllowsRedirectURI(request.RedirectURI), "accepted redirect_uri %q", request.RedirectURI)

		// RULE: The response type is one the server answers
		require.Contains(t, []string{"code", "token"}, request.ResponseType)

		// RULE: A challenge always names a method the server can check
		if request.CodeChallenge != "" {
			require.True(t, IsValidPKCEMethod(request.CodeChallengeMethod), "accepted method %q", request.CodeChallengeMethod)
		}

		// RULE: A public client's code is always bound to PKCE
		if (request.ResponseType == "code") && (secret == "") {
			require.NotEmpty(t, request.CodeChallenge)
		}

		// RULE: A client that registered scopes is granted none beyond them
		if client.Scopes.NotEmpty() {
			for _, requested := range request.Scopes() {
				require.Contains(t, client.Scopes, requested)
			}
		}
	})
}
