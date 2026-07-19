package unsplash

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestDisplayPhoto_EscapesCredits is the regression test for the Unsplash credit-line hardening:
// the photographer's username and display name come from the Unsplash API (external, untrusted),
// so they must be escaped — the display name via InnerText and the username via the href's Attr
// escaping — instead of being concatenated into a raw HTML string emitted through InnerHTML.
func TestDisplayPhoto_EscapesCredits(t *testing.T) {

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	ctx := echo.New().NewContext(request, recorder)

	photo := mapof.Any{
		"color":           "#336699",
		"alt_description": "a photo",
		"urls": mapof.Any{
			"regular": "https://images.example/regular.jpg",
			"small":   "https://images.example/small.jpg",
		},
		"user": mapof.Any{
			// A display name that tries to inject an element into the credit text.
			"name": `<img src=x onerror=alert(document.domain)>`,
			// A username that tries to break out of the href attribute.
			"username": `x" onmouseover=alert(1) x="`,
		},
	}

	require.Nil(t, displayPhoto(ctx, "TestApp", photo))

	body := recorder.Body.String()

	// The display name must be HTML-escaped in the link text, not rendered as a live element.
	require.NotContains(t, body, "<img src=x onerror", "photographer name broke out via InnerHTML")
	require.Contains(t, body, "&lt;img src=x onerror", "photographer name was not escaped")

	// The username must not break out of the href attribute: the injected quote must be escaped,
	// so a live `onmouseover=` handler never appears after a real closing quote.
	require.NotContains(t, body, `@x" onmouseover`, "username broke out of the href attribute")
	require.Contains(t, body, "&#34;", "username quote was not escaped")
}
