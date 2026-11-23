package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// PostResetCode must enforce the server-side password policy BEFORE loading the user or hashing the
// password, so that a crafted reset-code POST (which ignores the template's client-side minLength)
// cannot set a password that is too short. Steranko's default schema requires at least 8 characters;
// the rejection happens right after the passwords-match check and before any database work, so a
// zero Factory is enough -- steranko.New's dependencies are stored, never called, on this path.

// resetCodeRequest builds a urlencoded reset-code POST carrying a matching password pair.
func resetCodeRequest(password string) *steranko.Context {

	form := url.Values{
		"password":  {password},
		"password2": {password},
		"userId":    {"000000000000000000000000"},
		"code":      {"reset-code"},
	}

	request := httptest.NewRequest(http.MethodPost, "/signin/reset-code", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	return &steranko.Context{Context: echo.New().NewContext(request, recorder)}
}

// A reset-code submission with a password below the policy minimum must be rejected before any
// database work, not accepted and saved.
func TestPostResetCode_RejectsShortPassword(t *testing.T) {

	ctx := resetCodeRequest("a") // 1 character: far below steranko's 8-character default
	factory := &service.Factory{}

	// unusedSession panics on any database work, which the rejection must come before
	err := PostResetCode(ctx, factory, unusedSession{})

	require.NotNil(t, err, "a password below the policy minimum must be rejected")
}

// TestSignInAndReturnTo verifies that signing out to switch accounts returns to the whole page, query included,
// and never to another origin or back into a sign-in loop
func TestSignInAndReturnTo(t *testing.T) {

	// next returns where the sign-in page will send the User afterward
	next := func(value string) string {
		parsed, err := url.Parse(signInAndReturnTo(value))
		require.NoError(t, err)
		require.Equal(t, "/signin", parsed.Path)
		return parsed.Query().Get("next")
	}

	// The query string survives, which the OAuth consent page cannot do without
	require.Equal(t, "/oauth/authorize?client_id=x&scope=read%20write", next("/oauth/authorize?client_id=x&scope=read%20write"))

	// Another origin is reduced to a path on this one
	require.Equal(t, "/x?y=1", next("https://evil.example/x?y=1"))
	require.Equal(t, "/x", next("//evil.example/x"))

	// Loops and garbage go home
	require.Equal(t, "/", next("/signout"))
	require.Equal(t, "/", next("/signin?next=/x"))
	require.Equal(t, "/", next("%zz"))
}

// FuzzSignInAndReturnTo requires every destination to stay on this origin
func FuzzSignInAndReturnTo(f *testing.F) {

	f.Add("/oauth/authorize?client_id=x")
	f.Add("https://evil.example/x")
	f.Add("//evil.example/x")
	f.Add("/\\evil.example")
	f.Add("%zz")

	f.Fuzz(func(t *testing.T, value string) {

		parsed, err := url.Parse(signInAndReturnTo(value))
		require.NoError(t, err)
		require.Equal(t, "/signin", parsed.Path)

		destination := parsed.Query().Get("next")
		require.True(t, strings.HasPrefix(destination, "/"), "destination %q is not a path", destination)

		target, err := url.Parse(destination)
		if err != nil {
			return
		}

		require.Empty(t, target.Host, "destination %q leaves this origin", destination)
	})
}
