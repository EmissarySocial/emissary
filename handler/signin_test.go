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

	err := PostResetCode(ctx, factory, nil)

	require.NotNil(t, err, "a password below the policy minimum must be rejected")
}
