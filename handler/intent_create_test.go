package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestGetIntent_Create_EscapesContent is the regression test for the reported reflected XSS: the
// caller-supplied `content` query param is reflected into the post <textarea>. It must be
// HTML-escaped (InnerText), so a `</textarea>`-prefixed payload cannot break out of the textarea
// and execute in the Emissary origin.
func TestGetIntent_Create_EscapesContent(t *testing.T) {

	// The breakout payload from the bug report.
	const payload = `</textarea><img src=x onerror=alert(document.domain)>`

	request := httptest.NewRequest(http.MethodGet, "/@me/intent/create?content="+url.QueryEscape(payload), nil)
	recorder := httptest.NewRecorder()
	ctx := &steranko.Context{Context: echo.New().NewContext(request, recorder)}

	// GetIntent_Create renders from ctx + user only (no factory/session access on the GET path).
	user := model.NewUser()
	user.Username = "victim"
	user.ProfileURL = "https://example.com/@victim"

	require.Nil(t, GetIntent_Create(ctx, nil, nil, &user))

	body := recorder.Body.String()

	// The breakout markup must NOT appear raw...
	require.NotContains(t, body, "<img src=x onerror", "content broke out of the textarea (reflected XSS)")
	require.NotContains(t, body, "</textarea><img", "content closed the textarea early")

	// ...it must appear HTML-escaped instead.
	require.Contains(t, body, "&lt;/textarea&gt;", "content was not HTML-escaped")
	require.Contains(t, body, "&lt;img src=x onerror", "content was not HTML-escaped")
}
