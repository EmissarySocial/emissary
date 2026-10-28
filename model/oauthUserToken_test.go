package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestOAuthUserToken_IsCodeExpired verifies the expiration window of an authorization code
func TestOAuthUserToken_IsCodeExpired(t *testing.T) {

	now := time.Unix(1_700_000_000, 0)

	// The journal stores the creation time in Unix MILLISECONDS.
	grant := NewOAuthUserToken()

	t.Run("fresh code is not expired", func(t *testing.T) {
		grant.CreateDate = now.UnixMilli()
		require.False(t, grant.IsCodeExpired(now))
	})

	t.Run("code within the TTL is not expired", func(t *testing.T) {
		grant.CreateDate = now.Add(-OAuthCodeLifetime + time.Second).UnixMilli()
		require.False(t, grant.IsCodeExpired(now))
	})

	t.Run("code past the TTL is expired", func(t *testing.T) {
		grant.CreateDate = now.Add(-OAuthCodeLifetime - time.Second).UnixMilli()
		require.True(t, grant.IsCodeExpired(now))
	})
}

// TestOAuthUserToken_JSONResponse verifies the JSON body returned to an OAuth client
func TestOAuthUserToken_JSONResponse(t *testing.T) {

	grant := NewOAuthUserToken()
	grant.Token = "the.access.jwt"
	grant.RefreshToken = "grantid.1.the-refresh-secret"
	grant.Scopes = append(grant.Scopes, "read", "write")

	response := grant.JSONResponse()

	require.Equal(t, "the.access.jwt", response["access_token"])
	require.Equal(t, "Bearer", response["token_type"])
	require.Equal(t, "read write", response["scope"])
	require.Equal(t, "grantid.1.the-refresh-secret", response["refresh_token"])
	require.Equal(t, int(OAuthAccessTokenLifetime.Seconds()), response["expires_in"])
	require.Equal(t, 3600, response["expires_in"], "access token lifetime is 1 hour")
}

// TestOAuthUserToken_Toot verifies that a token renders as its Mastodon API equivalent
func TestOAuthUserToken_Toot(t *testing.T) {

	t.Run("with a refresh token, advertises expiry + refresh", func(t *testing.T) {
		grant := NewOAuthUserToken()
		grant.Token = "the.access.jwt"
		grant.RefreshToken = "grantid.1.secret"
		grant.Scopes = append(grant.Scopes, "read")

		toot := grant.Toot()
		require.Equal(t, "the.access.jwt", toot.AccessToken)
		require.Equal(t, "grantid.1.secret", toot.RefreshToken)
		require.Equal(t, int64(3600), toot.ExpiresIn)
	})

	t.Run("without a refresh token, omits expiry + refresh", func(t *testing.T) {
		grant := NewOAuthUserToken()
		grant.Token = "the.access.jwt"

		toot := grant.Toot()
		require.Empty(t, toot.RefreshToken)
		require.Zero(t, toot.ExpiresIn, "no expiry advertised when no refresh token is issued")
	})
}

// TestOAuthUserToken_MastodonAppLifetime confirms a Mastodon app's grant issues tokens with no
// expiry and advertises none, while other grants keep the standard lifetime
func TestOAuthUserToken_MastodonAppLifetime(t *testing.T) {

	standard := NewOAuthUserToken()
	standard.RefreshToken = "grantid.1.secret"
	require.Equal(t, OAuthAccessTokenLifetime, standard.AccessTokenLifetime())
	require.Equal(t, 3600, standard.JSONResponse()["expires_in"])

	mastodon := NewOAuthUserToken()
	mastodon.MastodonApp = true
	mastodon.RefreshToken = "grantid.1.secret"
	require.Equal(t, OAuthMastodonAppTokenLifetime, mastodon.AccessTokenLifetime())

	response := mastodon.JSONResponse()
	require.NotContains(t, response, "expires_in", "a token that never expires advertises no expiry")
	require.Equal(t, "grantid.1.secret", response["refresh_token"], "the refresh token is still offered")

	toot := mastodon.Toot()
	require.Zero(t, toot.ExpiresIn)
	require.Equal(t, "grantid.1.secret", toot.RefreshToken)
}
