package handler

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// drawConfirmation renders one intent's confirmation page for a transaction and the object it names
type drawConfirmation func(transaction camper.LikeIntent, object streams.Document) string

// checkConfirmationPage runs the checks that the Like, Dislike, and Announce confirmation pages all share
func checkConfirmationPage(t *testing.T, action string, draw drawConfirmation) {

	t.Helper()

	// RULE: The form's own fields must bind into what postIntent_Response reads, special characters and all
	t.Run("form posts what the POST reads", func(t *testing.T) {

		transaction := camper.LikeIntent{
			Object:    `https://remote.example/notes/1?a=1&b="2"`,
			OnSuccess: "https://host.example/done?x=1&y=<2>",
			OnCancel:  "(close)",
		}
		object := streams.NewDocument(map[string]any{"id": "https://remote.example/notes/1", "type": "Note"})

		body := draw(transaction, object)
		require.Regexp(t, `<form[^>]*method="POST"[^>]*action="`+regexp.QuoteMeta(action)+`"`, body)

		request := httptest.NewRequest(http.MethodPost, action, strings.NewReader(hiddenInputs(body).Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		ctx := &steranko.Context{Context: echo.New().NewContext(request, httptest.NewRecorder())}

		var posted camper.LikeIntent
		require.NoError(t, ctx.Bind(&posted))
		require.Equal(t, transaction, posted)
	})

	// Cancel goes through the continue page, to the caller's on-cancel or else to the User's own profile
	t.Run("cancel link", func(t *testing.T) {

		object := streams.NewDocument(map[string]any{"id": "https://remote.example/notes/1", "type": "Note"})

		body := draw(camper.LikeIntent{OnCancel: "https://host.example/back?x=1&y=2"}, object)
		require.Contains(t, body, `href="/@me/intent/continue?url=https%3A%2F%2Fhost.example%2Fback%3Fx%3D1%26y%3D2"`)

		body = draw(camper.LikeIntent{}, object)
		require.Contains(t, body, `href="/@me/intent/continue?url=%2F%40me"`)
	})

	// RULE: The author's name is escaped, and their icon is kept only when its URL is safe
	t.Run("author", func(t *testing.T) {

		authored := func(iconURL string) streams.Document {
			return streams.NewDocument(map[string]any{
				"id":   "https://remote.example/notes/1",
				"type": "Note",
				"attributedTo": map[string]any{
					"id":                "https://remote.example/users/bob",
					"type":              "Person",
					"name":              `Bob < & "co"`,
					"preferredUsername": "bob",
					"icon":              map[string]any{"type": "Image", "url": iconURL},
				},
			})
		}

		body := draw(camper.LikeIntent{}, authored("https://remote.example/bob.png"))
		require.Contains(t, body, `<div class="bold">Bob &lt; &amp; &#34;co&#34;</div>`)
		require.Contains(t, body, `<div class="text-gray">@bob@remote.example</div>`)
		require.Contains(t, body, `src="https://remote.example/bob.png"`)

		body = draw(camper.LikeIntent{}, authored("javascript:alert(1)"))
		require.NotContains(t, strings.ToLower(body), "javascript:", "an unsafe icon URL reached the page")
	})

	// RULE: The object's content is remote HTML, so the page keeps its text and drops its scripts
	t.Run("content is sanitized", func(t *testing.T) {

		object := streams.NewDocument(map[string]any{
			"id":      "https://remote.example/notes/1",
			"type":    "Note",
			"content": `<p>Hello</p><script>alert(1)</script><img src=x onerror=alert(2)>`,
		})

		body := draw(camper.LikeIntent{}, object)
		require.Contains(t, body, "<p>Hello</p>")
		require.NotContains(t, body, "alert(1)", "a remote script reached the page")
		require.NotContains(t, body, "onerror", "a remote event handler reached the page")
	})
}

// hiddenInputs returns the name and value of every hidden input on a page, as a browser would post them
func hiddenInputs(body string) url.Values {

	result := url.Values{}
	nameAttribute := regexp.MustCompile(`name="([^"]*)"`)
	valueAttribute := regexp.MustCompile(`value="([^"]*)"`)

	for _, tag := range regexp.MustCompile(`<input[^>]*>`).FindAllString(body, -1) {

		if !strings.Contains(tag, `type="hidden"`) {
			continue
		}

		name := nameAttribute.FindStringSubmatch(tag)
		if name == nil {
			continue
		}

		value := ""
		if match := valueAttribute.FindStringSubmatch(tag); match != nil {
			value = html.UnescapeString(match[1])
		}

		result.Add(html.UnescapeString(name[1]), value)
	}

	// What the browser sees is what the server gets
	return result
}
