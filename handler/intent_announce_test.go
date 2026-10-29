package handler

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// announcePage renders the Announce confirmation page for one object, with no factory or network
func announcePage(transaction camper.AnnounceIntent, object streams.Document) string {

	request := httptest.NewRequest(http.MethodGet, "/@me/intent/announce", nil)
	ctx := &steranko.Context{Context: echo.New().NewContext(request, httptest.NewRecorder())}

	user := model.NewUser()
	user.Username = "sharer"

	return getIntent_Announce(ctx, &user, service.Icons{}, transaction, object)
}

// The Announce intent shares postIntent_Response, so it must reject a bogus target as Like does.
func TestPostIntent_Announce_RejectsInvalidTarget(t *testing.T) {

	ctx := intentRequest("not-a-url")
	factory := &service.Factory{}
	user := model.NewUser()

	// unusedSession panics on any database work, which the rejection must come before
	err := PostIntent_Announce(ctx, factory, unusedSession{}, &user)

	require.NotNil(t, err, "an invalid reaction target must be rejected")
}

// The confirmation form must post the object back, because its action carries no query string.
func TestGetIntent_Announce_FormPostsTheObject(t *testing.T) {

	const objectURL = "https://remote.example/notes/1"

	transaction := camper.AnnounceIntent{Object: objectURL, OnSuccess: "(close)", OnCancel: "(close)"}
	object := streams.NewDocument(map[string]any{"id": objectURL, "type": "Note", "content": "Hello there"})

	body := announcePage(transaction, object)

	require.Regexp(t, `<form[^>]*method="POST"[^>]*action="/@me/intent/announce"`, body)
	require.Regexp(t, `<input[^>]*name="object"[^>]*value="`+regexp.QuoteMeta(objectURL)+`"`, body)
	require.Regexp(t, `<input[^>]*name="on-success"[^>]*value="\(close\)"`, body)
	require.Contains(t, body, "Hello there", "the page shows the object being shared")
	require.Contains(t, body, "Share This")
}

// A remote object's name and a caller's object URL must reach the page escaped.
func TestGetIntent_Announce_EscapesRemoteValues(t *testing.T) {

	const payload = `"><img src=x onerror=alert(document.domain)>`

	transaction := camper.AnnounceIntent{Object: "https://remote.example/notes/1?q=" + payload}
	object := streams.NewDocument(map[string]any{"id": "https://remote.example/notes/1", "type": "Note", "name": payload})

	body := announcePage(transaction, object)

	require.NotContains(t, body, "<img src=x onerror", "a remote value broke out of its element or attribute")
}

// A remote name is escaped by the page itself, not only stripped of tags by hannibal on its way in.
func TestGetIntent_Announce_EscapesTheName(t *testing.T) {

	object := streams.NewDocument(map[string]any{"id": "https://remote.example/notes/1", "type": "Note", "name": `a < b & "c"`})

	body := announcePage(camper.AnnounceIntent{}, object)

	require.Contains(t, body, `<div class="margin-top-none text-lg bold">a &lt; b &amp; &#34;c&#34;</div>`)
}
