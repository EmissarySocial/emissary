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
				if requested != "" {
					require.Contains(t, client.Scopes, requested)
				}
			}
		}

		// RULE: An implicit grant always names a scope, because it always returns an access token
		if request.ResponseType == "token" {
			require.True(t, request.hasScope())
		}
	})
}

// TestOAuthAuthorizationRequest_Scope verifies the default scope, the scopes a client is limited to,
// and that an implicit grant must name one
func TestOAuthAuthorizationRequest_Scope(t *testing.T) {

	registered := NewOAuthClient()
	registered.ClientSecret = "secret"
	registered.RedirectURIs = append(registered.RedirectURIs, "https://app.example.com/callback")
	registered.Scopes = append(registered.Scopes, "read", "write")

	// A URL client that publishes no metadata, as the OAuthClient service creates it
	identityOnly := NewOAuthClient()
	identityOnly.ClientURL = "https://app.example.com/"
	identityOnly.Scopes = append(identityOnly.Scopes, OAuthUserTokenScopeProfile, OAuthUserTokenScopeEmail)

	const challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	urlRequest := func(responseType string, scope string) OAuthAuthorizationRequest {
		return OAuthAuthorizationRequest{ResponseType: responseType, Scope: scope, RedirectURI: "https://app.example.com/callback", CodeChallenge: challenge, CodeChallengeMethod: PKCEMethodS256}
	}

	t.Run("a registered client that names no scope gets the scopes it registered", func(t *testing.T) {
		request := OAuthAuthorizationRequest{ResponseType: "code"}
		require.NoError(t, request.Validate(registered))
		require.Equal(t, "read write", request.Scope)
	})

	t.Run("a URL client that names no scope asks for none", func(t *testing.T) {
		request := urlRequest("code", "")
		require.NoError(t, request.Validate(identityOnly))
		require.Empty(t, request.Scope)
	})

	t.Run("a client with no metadata may ask who the User is", func(t *testing.T) {
		for _, scope := range []string{"profile", "email", "profile email"} {
			request := urlRequest("code", scope)
			require.NoError(t, request.Validate(identityOnly), "scope %q", scope)
		}
	})

	t.Run("a client with no metadata may ask for nothing else", func(t *testing.T) {
		for _, scope := range []string{"read", "write", "profile write", OAuthUserTokenScopeActivityPubPortability, "made-up"} {
			request := urlRequest("code", scope)
			require.True(t, derp.IsBadRequest(request.Validate(identityOnly)), "scope %q", scope)
		}
	})

	t.Run("an implicit grant must name a scope", func(t *testing.T) {
		for _, scope := range []string{"", " ", "  "} {
			request := urlRequest("token", scope)
			require.True(t, derp.IsBadRequest(request.Validate(identityOnly)), "scope %q", scope)
		}

		request := urlRequest("token", "profile")
		require.NoError(t, request.Validate(identityOnly))

		// A registered client's default scope counts
		request = OAuthAuthorizationRequest{ResponseType: "token"}
		require.NoError(t, request.Validate(registered))
	})
}

// TestOAuthAuthorizationRequest_IsDenied pins the one value that refuses a request
func TestOAuthAuthorizationRequest_IsDenied(t *testing.T) {

	require.True(t, OAuthAuthorizationRequest{Decision: "deny"}.IsDenied())

	for _, decision := range []string{"", "allow", "DENY", " deny", "deny ", "true"} {
		require.False(t, OAuthAuthorizationRequest{Decision: decision}.IsDenied(), "decision %q", decision)
	}
}
