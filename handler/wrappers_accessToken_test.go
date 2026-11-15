package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// testTokenKey signs the tokens below. Detaching never checks a signature, so any key will do.
var testTokenKey = []byte("wrappers_accessToken_test")

// mintAccessToken returns a signed JWT in the shape OAuthUserToken.JWT mints for an OAuth client
func mintAccessToken(t *testing.T) string {

	t.Helper()
	userID := primitive.NewObjectID()

	return signClaims(t, jwt.MapClaims{
		"A":   true,
		"U":   userID,
		"S":   "read",
		"K":   primitive.NewObjectID(),
		"sub": userID.Hex(),
		"iat": jwt.NewNumericDate(time.Now()),
	})
}

// mintLegacyAccessToken returns a signed JWT in the shape minted before grants were recorded: APIUser,
// no grant ID, and no expiry
func mintLegacyAccessToken(t *testing.T) string {

	t.Helper()
	userID := primitive.NewObjectID()

	return signClaims(t, jwt.MapClaims{"A": true, "U": userID, "S": "read"})
}

// mintSession returns a signed JWT in the shape SterankoUserService mints for a signed-in person
func mintSession(t *testing.T) string {

	t.Helper()
	authorization := model.NewAuthorization()
	authorization.UserID = primitive.NewObjectID()
	authorization.DomainOwner = true
	authorization.Revalidate = time.Now().Unix()
	authorization.Subject = authorization.UserID.Hex()

	return signClaims(t, &authorization)
}

// signClaims returns the claims as a signed JWT
func signClaims(t *testing.T, claims jwt.Claims) string {

	t.Helper()
	result, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testTokenKey)
	require.NoError(t, err)

	return result
}

// TestIsAccessToken confirms the claim shapes that count as an access token, and those that do not
func TestIsAccessToken(t *testing.T) {
	require.True(t, isAccessToken(mintAccessToken(t)), "a token with APIUser and a grant ID")
	require.True(t, isAccessToken(mintLegacyAccessToken(t)), "a legacy token with APIUser and no grant ID")
	require.False(t, isAccessToken(mintSession(t)), "a session carries neither marker")
	require.False(t, isAccessToken(""), "no credential")
	require.False(t, isAccessToken("not-a-jwt"), "a bare string")
	require.False(t, isAccessToken("a.b.c"), "three segments that do not decode")
	require.False(t, isAccessToken(mintAccessToken(t)+".extra"), "too many segments")
}

// TestIsAccessToken_UnsignedShape confirms that the decision reads the claims alone, so a forged token
// shaped like an access token is removed as well, and nothing else ever trusts it
func TestIsAccessToken_UnsignedShape(t *testing.T) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"A": true}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	require.True(t, isAccessToken(token))
}

// TestBearerToken confirms the header forms steranko accepts are reduced to the same credential
func TestBearerToken(t *testing.T) {
	require.Equal(t, "abc", bearerToken("Bearer abc"))
	require.Equal(t, "abc", bearerToken("bearer abc"), "the scheme is case-insensitive")
	require.Equal(t, "abc", bearerToken("Bearer   abc  "), "padding around the token is ignored")
	require.Equal(t, "abc", bearerToken("abc"), "a bare token has no scheme")
	require.Equal(t, "Basic abc", bearerToken("Basic abc"), "another scheme is left whole")
	require.Equal(t, "", bearerToken(""))
}

// TestDetachAccessTokens_Header confirms that an access token leaves the Authorization header in every
// form steranko would read it, while a session and other schemes stay
func TestDetachAccessTokens_Header(t *testing.T) {

	withHeader := func(value string) *http.Request {
		request := httptest.NewRequest(http.MethodGet, "https://local.example/@me", nil)
		request.Header.Set("Authorization", value)
		return request
	}

	accessToken := mintAccessToken(t)
	session := mintSession(t)

	request := withHeader("Bearer " + accessToken)
	detachAccessTokens(request)
	require.Empty(t, request.Header.Get("Authorization"), "an access token as a Bearer header is removed")

	request = withHeader("bearer " + accessToken)
	detachAccessTokens(request)
	require.Empty(t, request.Header.Get("Authorization"), "the scheme is case-insensitive")

	request = withHeader(accessToken)
	detachAccessTokens(request)
	require.Empty(t, request.Header.Get("Authorization"), "a bare access token is removed")

	request = withHeader("Bearer " + mintLegacyAccessToken(t))
	detachAccessTokens(request)
	require.Empty(t, request.Header.Get("Authorization"), "a legacy access token is removed")

	request = withHeader("Bearer " + session)
	detachAccessTokens(request)
	require.Equal(t, "Bearer "+session, request.Header.Get("Authorization"), "a session JWT is left where it was")

	request = withHeader("Basic dXNlcjpwYXNz")
	detachAccessTokens(request)
	require.Equal(t, "Basic dXNlcjpwYXNz", request.Header.Get("Authorization"), "another scheme is left alone")

	request = httptest.NewRequest(http.MethodGet, "https://local.example/@me", nil)
	detachAccessTokens(request)
	require.Empty(t, request.Header.Values("Authorization"), "a request with no credential gains none")
	require.Empty(t, request.Header.Values("Cookie"))
}

// TestDetachAccessTokens_Cookies confirms that an access token leaves every cookie steranko reads, whatever
// its name, while the cookies around it survive
func TestDetachAccessTokens_Cookies(t *testing.T) {

	accessToken := mintAccessToken(t)

	// Every cookie name steranko reads: the TLS and plain-HTTP session cookies, and their masquerade backups
	for _, name := range []string{"Authorization", "__Host-Authorization", "Authorization-backup", "__Host-Authorization-backup"} {

		t.Run(name, func(t *testing.T) {

			request := httptest.NewRequest(http.MethodPost, "https://local.example/signout", nil)
			request.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
			request.AddCookie(&http.Cookie{Name: name, Value: accessToken})
			request.AddCookie(&http.Cookie{Name: "lang", Value: "en-US"})

			detachAccessTokens(request)

			_, err := request.Cookie(name)
			require.ErrorIs(t, err, http.ErrNoCookie, "the access token is removed")

			theme, err := request.Cookie("theme")
			require.NoError(t, err)
			require.Equal(t, "dark", theme.Value, "the cookie before it survives")

			lang, err := request.Cookie("lang")
			require.NoError(t, err)
			require.Equal(t, "en-US", lang.Value, "the cookie after it survives")
		})
	}
}

// TestDetachAccessTokens_LegacyCookie confirms a grant-less token in the session cookie is removed too
func TestDetachAccessTokens_LegacyCookie(t *testing.T) {

	request := httptest.NewRequest(http.MethodGet, "https://local.example/@me", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-Authorization", Value: mintLegacyAccessToken(t)})

	detachAccessTokens(request)

	require.Empty(t, request.Header.Values("Cookie"), "the only cookie was the token, so the header goes with it")
}

// TestDetachAccessTokens_CookiesUntouched confirms that a request carrying no access token keeps its Cookie
// header byte for byte, so ordinary sessions are never rewritten
func TestDetachAccessTokens_CookiesUntouched(t *testing.T) {

	request := httptest.NewRequest(http.MethodGet, "https://local.example/@me", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-Authorization", Value: mintSession(t)})
	request.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
	before := request.Header.Get("Cookie")

	detachAccessTokens(request)

	require.Equal(t, before, request.Header.Get("Cookie"))
}

// TestWithoutAccessTokens confirms the next handler sees a request with every access token already gone,
// and that its answer comes back unchanged
func TestWithoutAccessTokens(t *testing.T) {

	e := echo.New()
	request := httptest.NewRequest(http.MethodGet, "https://local.example/@me", nil)
	request.Header.Set("Authorization", "Bearer "+mintAccessToken(t))
	request.AddCookie(&http.Cookie{Name: "__Host-Authorization", Value: mintAccessToken(t)})
	ctx := e.NewContext(request, httptest.NewRecorder())

	expected := errors.New("from the next handler")
	called := false

	handler := withoutAccessTokens(func(ctx echo.Context) error {
		called = true
		require.Empty(t, ctx.Request().Header.Get("Authorization"))
		require.Empty(t, ctx.Request().Header.Values("Cookie"))
		return expected
	})

	require.ErrorIs(t, handler(ctx), expected)
	require.True(t, called)
}
