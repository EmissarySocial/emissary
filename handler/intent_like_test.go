package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// The Like/Dislike intent POST handlers forward the caller-supplied `object` to
// service.Response.SetResponse, which validates the reaction target. These tests confirm the
// handlers surface that rejection (they do NOT store a Response for a bogus target) rather than
// returning 200.
//
// A malformed URL is rejected by SetResponse's cheap syntactic gate BEFORE any object load or DB
// read, so a zero Factory + a bare Response service is enough: the network/Mongo stack is never
// reached. (The unresolvable-but-well-formed case needs the ActivityStream loader, which is only
// wired up by Factory.Refresh against a live domain, so it is covered by the service-level tests.)

// intentRequest builds a urlencoded POST carrying the reaction target.
func intentRequest(object string) *steranko.Context {

	form := url.Values{"object": {object}}

	request := httptest.NewRequest(http.MethodPost, "/@me/intent/like", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	return &steranko.Context{Context: echo.New().NewContext(request, recorder)}
}

// likePage renders the Like confirmation page for one object, with no factory or network
func likePage(transaction camper.LikeIntent, object streams.Document) string {

	request := httptest.NewRequest(http.MethodGet, "/@me/intent/like", nil)
	ctx := &steranko.Context{Context: echo.New().NewContext(request, httptest.NewRecorder())}

	user := model.NewUser()
	user.Username = "liker"

	return getIntent_Like(ctx, &user, service.Icons{}, transaction, object)
}

// A Like intent on a malformed target must return an error, not a 200 with a stored Response.
func TestPostIntent_Like_RejectsInvalidTarget(t *testing.T) {

	ctx := intentRequest("not-a-url")
	factory := &service.Factory{}
	user := model.NewUser()

	// unusedSession panics on any database work, which the rejection must come before
	err := PostIntent_Like(ctx, factory, unusedSession{}, &user)

	require.NotNil(t, err, "an invalid reaction target must be rejected")
}

// The confirmation form must post the object back, because its action carries no query string.
func TestGetIntent_Like_FormPostsTheObject(t *testing.T) {

	const objectURL = "https://remote.example/notes/1"

	transaction := camper.LikeIntent{Object: objectURL, OnSuccess: "(close)", OnCancel: "(close)"}
	object := streams.NewDocument(map[string]any{"id": objectURL, "type": "Note", "content": "Hello there"})

	body := likePage(transaction, object)

	require.Regexp(t, `<form[^>]*method="POST"[^>]*action="/@me/intent/like"`, body)
	require.Regexp(t, `<input[^>]*name="object"[^>]*value="`+regexp.QuoteMeta(objectURL)+`"`, body)
	require.Regexp(t, `<input[^>]*name="on-success"[^>]*value="\(close\)"`, body)
	require.Contains(t, body, "Hello there", "the page shows the object being liked")
	require.Contains(t, body, "Like This")
}

// A remote object's name and a caller's object URL must reach the page escaped.
func TestGetIntent_Like_EscapesRemoteValues(t *testing.T) {

	const payload = `"><img src=x onerror=alert(document.domain)>`

	transaction := camper.LikeIntent{Object: "https://remote.example/notes/1?q=" + payload}
	object := streams.NewDocument(map[string]any{"id": "https://remote.example/notes/1", "type": "Note", "name": payload})

	body := likePage(transaction, object)

	require.NotContains(t, body, "<img src=x onerror", "a remote value broke out of its element or attribute")
}

// A remote name is escaped by the page itself, not only stripped of tags by hannibal on its way in.
func TestGetIntent_Like_EscapesTheName(t *testing.T) {

	object := streams.NewDocument(map[string]any{"id": "https://remote.example/notes/1", "type": "Note", "name": `a < b & "c"`})

	body := likePage(camper.LikeIntent{}, object)

	require.Contains(t, body, `<div class="margin-top-none text-lg bold">a &lt; b &amp; &#34;c&#34;</div>`)
}

// The Like page passes every check a confirmation page shares (intent_confirmation_test.go).
func TestGetIntent_Like_ConfirmationPage(t *testing.T) {
	checkConfirmationPage(t, "/@me/intent/like", likePage)
}
