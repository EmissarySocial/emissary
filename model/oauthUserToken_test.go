package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// TestOAuthUserToken_RedirectURI verifies that a grant records the redirect_uri its code was issued to,
// and keeps it through storage
func TestOAuthUserToken_RedirectURI(t *testing.T) {

	const redirectURI = "https://client.example/callback?keep=1"

	t.Run("records the redirect_uri", func(t *testing.T) {
		grant := NewOAuthUserToken()
		grant.SetRedirectURI(redirectURI)
		require.Equal(t, redirectURI, grant.RedirectURI())
	})

	t.Run("a blank value records nothing", func(t *testing.T) {
		grant := NewOAuthUserToken()
		grant.SetRedirectURI("")
		require.Empty(t, grant.RedirectURI())
		require.NotContains(t, grant.Data, oauthUserTokenDataRedirectURI)
	})

	t.Run("a grant with no data map", func(t *testing.T) {
		grant := OAuthUserToken{}
		require.Empty(t, grant.RedirectURI())

		grant.SetRedirectURI(redirectURI)
		require.Equal(t, redirectURI, grant.RedirectURI())
	})

	t.Run("survives storage beside the PKCE challenge", func(t *testing.T) {
		grant := NewOAuthUserToken()
		grant.SetPKCEChallenge(strings.Repeat("v", 43), PKCEMethodPlain)
		grant.SetRedirectURI(redirectURI)

		stored, err := bson.Marshal(grant)
		require.NoError(t, err)

		loaded := NewOAuthUserToken()
		require.NoError(t, bson.Unmarshal(stored, &loaded))
		require.Equal(t, redirectURI, loaded.RedirectURI())
		require.True(t, loaded.HasPKCEChallenge())
	})

	t.Run("a grant stored before it was recorded", func(t *testing.T) {
		stored, err := bson.Marshal(NewOAuthUserToken())
		require.NoError(t, err)

		loaded := NewOAuthUserToken()
		require.NoError(t, bson.Unmarshal(stored, &loaded))
		require.Empty(t, loaded.RedirectURI())
	})
}

// TestOAuthUserToken_VerifyRedirectURI verifies which token requests name the redirect_uri a
// grant's code was issued to (RFC 6749 §4.1.3, IndieAuth §5.3.1)
func TestOAuthUserToken_VerifyRedirectURI(t *testing.T) {

	const issuedTo = "https://client.example/callback"

	registered := NewOAuthClient()
	urlClient := NewOAuthClient()
	urlClient.ClientURL = "https://client.example/"

	grant := NewOAuthUserToken()
	grant.SetRedirectURI(issuedTo)

	t.Run("the same redirect_uri", func(t *testing.T) {
		require.NoError(t, grant.VerifyRedirectURI(registered, issuedTo))
		require.NoError(t, grant.VerifyRedirectURI(urlClient, issuedTo))
	})

	t.Run("a different redirect_uri", func(t *testing.T) {
		for _, presented := range []string{
			"https://evil.example/callback",
			"https://client.example/callback/",
			"https://client.example/callback?x=1",
			"https://CLIENT.example/callback",
			"http://client.example/callback",
			" " + issuedTo,
		} {
			require.Error(t, grant.VerifyRedirectURI(registered, presented), presented)
			require.Error(t, grant.VerifyRedirectURI(urlClient, presented), presented)
		}
	})

	t.Run("no redirect_uri from a registered client", func(t *testing.T) {
		require.NoError(t, grant.VerifyRedirectURI(registered, ""), "RFC 6749 requires it only when the authorization request carried it")
	})

	t.Run("no redirect_uri from a URL client", func(t *testing.T) {
		require.Error(t, grant.VerifyRedirectURI(urlClient, ""), "IndieAuth requires it")
	})

	t.Run("a grant issued before the redirect_uri was recorded", func(t *testing.T) {
		legacy := NewOAuthUserToken()
		require.NoError(t, legacy.VerifyRedirectURI(urlClient, "https://anything.example/"))
		require.NoError(t, legacy.VerifyRedirectURI(urlClient, ""))
	})

	t.Run("a grant with no data map", func(t *testing.T) {
		legacy := OAuthUserToken{}
		require.NoError(t, legacy.VerifyRedirectURI(registered, issuedTo))
	})
}

// TestOAuthUserToken_IssuesAccessToken verifies that only a grant carrying a scope gets an access token
func TestOAuthUserToken_IssuesAccessToken(t *testing.T) {

	issues := func(scopes ...string) bool {
		grant := NewOAuthUserToken()
		grant.Scopes = scopes
		return grant.IssuesAccessToken()
	}

	require.False(t, issues(), "no scopes")
	require.False(t, issues(""), "an authorization request with no scope stores one empty scope")
	require.False(t, issues("", ""), "a scope of only separators")
	require.False(t, OAuthUserToken{}.IssuesAccessToken(), "a nil slice")

	require.True(t, issues("profile"))
	require.True(t, issues("read", "write"))
	require.True(t, issues("", "email"), "a doubled separator still carries a scope")
}

// TestOAuthUserToken_HasScope verifies scope lookups on a grant
func TestOAuthUserToken_HasScope(t *testing.T) {

	grant := NewOAuthUserToken()
	grant.Scopes = append(grant.Scopes, "profile", "read:follows")

	require.True(t, grant.HasScope(OAuthUserTokenScopeProfile))
	require.True(t, grant.HasScope("read:follows"))
	require.False(t, grant.HasScope(OAuthUserTokenScopeEmail))
	require.False(t, grant.HasScope("read"), "a child scope does not grant its parent")
	require.False(t, grant.HasScope("Profile"), "scopes are case-sensitive")
	require.False(t, OAuthUserToken{}.HasScope(OAuthUserTokenScopeProfile))
}

// TestOAuthUserToken_TokenResponse verifies the token endpoint's answer for each kind of grant and
// User (IndieAuth §5.3)
func TestOAuthUserToken_TokenResponse(t *testing.T) {

	const profileURL = "https://example.com/@000000000000000000000001"

	// newPrimaryUser returns the public primary user of example.com
	newPrimaryUser := func() User {
		user := NewUser()
		user.IsOwner = true
		user.IsPrimary = true
		user.IsPublic = true
		user.ProfileURL = profileURL
		user.DisplayName = "Ben Pate"
		user.EmailAddress = "ben@example.com"
		user.IconID = primitive.NewObjectID()
		return user
	}

	// newGrant returns a redeemed grant with the provided scopes and freshly minted tokens
	newGrant := func(scopes ...string) OAuthUserToken {
		grant := NewOAuthUserToken()
		grant.Scopes = scopes
		grant.Token = "the.access.jwt"
		grant.RefreshToken = "grantid.1.the-refresh-secret"
		return grant
	}

	t.Run("no scope answers with me alone", func(t *testing.T) {
		grant := newGrant("")
		user := newPrimaryUser()

		require.True(t, grant.IsRedeemableFor(user))
		require.Equal(t, map[string]any{"me": profileURL}, grant.TokenResponse(user))
	})

	t.Run("no scope for a User the domain does not vouch for answers nothing", func(t *testing.T) {
		grant := newGrant("")
		user := newPrimaryUser()
		user.IsPrimary = false

		require.False(t, grant.IsRedeemableFor(user))
		require.Empty(t, grant.TokenResponse(user))
	})

	t.Run("profile releases the name, url, and photo", func(t *testing.T) {
		grant := newGrant(OAuthUserTokenScopeProfile)
		user := newPrimaryUser()
		response := grant.TokenResponse(user)

		require.Equal(t, profileURL, response["me"])
		require.Equal(t, map[string]any{
			"url":   profileURL,
			"name":  "Ben Pate",
			"photo": user.ActivityPubIconURL(),
		}, response["profile"])

		require.Equal(t, "the.access.jwt", response["access_token"], "a scope gets its tokens")
		require.Equal(t, "grantid.1.the-refresh-secret", response["refresh_token"])
		require.Equal(t, "profile", response["scope"])
	})

	t.Run("email is released inside the profile", func(t *testing.T) {
		grant := newGrant(OAuthUserTokenScopeProfile, OAuthUserTokenScopeEmail)
		profile, ok := grant.TokenResponse(newPrimaryUser())["profile"].(map[string]any)

		require.True(t, ok)
		require.Equal(t, "ben@example.com", profile["email"])
	})

	t.Run("email without profile releases no profile", func(t *testing.T) {
		grant := newGrant(OAuthUserTokenScopeEmail)
		response := grant.TokenResponse(newPrimaryUser())

		require.Equal(t, profileURL, response["me"])
		require.NotContains(t, response, "profile")
		require.NotContains(t, fmt.Sprint(response), "ben@example.com")
	})

	t.Run("blank profile fields are left out", func(t *testing.T) {
		grant := newGrant(OAuthUserTokenScopeProfile, OAuthUserTokenScopeEmail)
		user := newPrimaryUser()
		user.DisplayName = ""
		user.EmailAddress = ""
		user.IconID = primitive.NilObjectID

		require.Equal(t, map[string]any{"url": profileURL}, grant.TokenResponse(user)["profile"])
	})

	t.Run("a User the domain does not vouch for gets tokens and no identity", func(t *testing.T) {
		grant := newGrant("read", OAuthUserTokenScopeProfile, OAuthUserTokenScopeEmail)
		user := newPrimaryUser()
		user.IsPrimary = false

		response := grant.TokenResponse(user)
		require.True(t, grant.IsRedeemableFor(user))
		require.Equal(t, grant.JSONResponse()["access_token"], response["access_token"])
		require.NotContains(t, response, "me")
		require.NotContains(t, response, "profile")
	})

	t.Run("read without profile releases no profile", func(t *testing.T) {
		response := newGrant("read").TokenResponse(newPrimaryUser())
		require.Equal(t, profileURL, response["me"])
		require.NotContains(t, response, "profile")
	})
}
