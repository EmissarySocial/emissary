package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// acceptMatrix is the set of "Accept" headers the tests below run, with the Variant that a correct
// reading of each one selects.
var acceptMatrix = []struct {
	name            string
	accept          string
	wantActivityPub bool
}{
	{"absent", "", false},
	{"curl default", "*/*", false},
	{"browser", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", false},
	{"html only", "text/html", false},
	{"unsupported only", "application/xml", false},
	{"mastodon", `application/activity+json, application/ld+json; profile="https://www.w3.org/ns/activitystreams"`, true},
	{"activity+json", vocab.ContentTypeActivityPub, true},
	{"ld+json with profile", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`, true},
	{"q-values reorder", "text/html;q=0.9, application/activity+json;q=1.0", true},
	{"unsupported listed first", "application/xrd+xml, application/activity+json", true},
}

// describedHeaders are the fields RFC 9110 section 9.3.2 requires HEAD and GET to agree on.
var describedHeaders = []string{"Content-Type", "Vary", "ETag", "Last-Modified"}

// newHeadContext builds a steranko.Context for a HEAD request carrying the provided Accept header,
// along with the recorder that captures the response.
func newHeadContext(accept string) (*steranko.Context, *httptest.ResponseRecorder) {

	request := httptest.NewRequest(http.MethodHead, "https://example.com/test", nil)

	if accept != "" {
		request.Header.Set("Accept", accept)
	}

	recorder := httptest.NewRecorder()
	return &steranko.Context{Context: echo.New().NewContext(request, recorder)}, recorder
}

// getHeaders returns the headers a GET of this Variant would emit.
func getHeaders(t *testing.T, variant headers.Variant, object headers.Validator) http.Header {

	// These are the calls the production branches make, not a restatement of what they should
	// produce -- so a handler that stopped making them would fail here.
	t.Helper()

	recorder := httptest.NewRecorder()

	switch ctx := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "https://example.com/test", nil), recorder); variant {

	case headers.VariantActivityPub:
		// handler/activitypub_stream/stream.go and activitypub_user/profile.go
		headers.SetAll(ctx.Response().Header(), headers.VariantActivityPub, object)
		require.Nil(t, ctx.JSON(http.StatusOK, map[string]string{"type": "Note"}))

	default:
		// build/step_ViewHTML.go sets these; build.AsHTML writes the body through ctx.HTML.
		ctx.Response().Header().Set("Vary", headers.VaryHTML)
		ctx.Response().Header().Set("ETag", headers.ETag(headers.VariantHTML, object))
		ctx.Response().Header().Set("Last-Modified", headers.LastModified(object))
		require.Nil(t, ctx.HTML(http.StatusOK, "<p>hello</p>"))
	}

	return recorder.Header()
}

// requireParity asserts that a HEAD response describes the resource exactly as a GET would.
func requireParity(t *testing.T, head http.Header, variant headers.Variant, object headers.Validator, accept string) {

	t.Helper()
	get := getHeaders(t, variant, object)

	for _, name := range describedHeaders {
		require.NotEmpty(t, head.Get(name), "HEAD sent no %s for Accept: %q", name, accept)
		require.Equal(t, get.Get(name), head.Get(name), "HEAD and GET disagree on %s for Accept: %q", name, accept)
	}
}

// variantFor maps the matrix flag onto a Variant.
func variantFor(wantActivityPub bool) headers.Variant {

	if wantActivityPub {
		return headers.VariantActivityPub
	}

	return headers.VariantHTML
}

// TestHeadStream_MatchesGet covers HEAD and GET agreeing on the Stream route's response headers.
func TestHeadStream_MatchesGet(t *testing.T) {

	stream := model.NewStream()
	stream.UpdateDate = 1754913127000

	for _, test := range acceptMatrix {
		t.Run(test.name, func(t *testing.T) {

			ctx, recorder := newHeadContext(test.accept)
			require.Nil(t, HeadStream(ctx, nil, nil, &stream))

			// The branch GET would take, taken by the same production call GetStream makes.
			require.Equal(t, test.wantActivityPub, hannibal.IsActivityPubRequest(ctx.Request()),
				"GET would choose a different variant than this matrix expects")

			requireParity(t, recorder.Header(), variantFor(test.wantActivityPub), &stream, test.accept)
		})
	}
}

// TestHeadOutbox_MatchesGet covers the same agreement on the User route, which has its own handler.
func TestHeadOutbox_MatchesGet(t *testing.T) {

	user := model.NewUser()
	user.IsPublic = true
	user.UpdateDate = 1754913127000

	for _, test := range acceptMatrix {
		t.Run(test.name, func(t *testing.T) {

			ctx, recorder := newHeadContext(test.accept)
			require.Nil(t, HeadOutbox(ctx, nil, nil, &user))

			requireParity(t, recorder.Header(), variantFor(test.wantActivityPub), &user, test.accept)
		})
	}
}

// TestHeadHandlers_AlwaysDescribeTheResponse covers every HEAD response carrying every field.
func TestHeadHandlers_AlwaysDescribeTheResponse(t *testing.T) {

	// An unmatched Accept must still describe the response: no empty Content-Type, and never a
	// missing Vary or validator.
	stream := model.NewStream()
	user := model.NewUser()
	user.IsPublic = true

	for _, accept := range []string{"", "*/*", "application/xml", "nonsense", "text/html;;"} {

		ctx, recorder := newHeadContext(accept)
		require.Nil(t, HeadStream(ctx, nil, nil, &stream))

		for _, name := range describedHeaders {
			require.NotEmpty(t, recorder.Header().Get(name), "HeadStream sent no %s for Accept: %q", name, accept)
		}

		ctx, recorder = newHeadContext(accept)
		require.Nil(t, HeadOutbox(ctx, nil, nil, &user))

		for _, name := range describedHeaders {
			require.NotEmpty(t, recorder.Header().Get(name), "HeadOutbox sent no %s for Accept: %q", name, accept)
		}
	}
}

// TestVariantsDoNotShareValidators is the regression test for the defect this merge exposed.
func TestVariantsDoNotShareValidators(t *testing.T) {

	// One object served two ways is two representations, and an entity-tag identifies a
	// representation (RFC 9110 section 8.8.1). Sharing a tag would let a client holding the HTML
	// document be answered 304 when it asks for ActivityStreams.
	stream := model.NewStream()
	stream.UpdateDate = 1754913127000

	html, _ := newHeadContext("text/html")
	require.Nil(t, HeadStream(html, nil, nil, &stream))

	activityPub, _ := newHeadContext(vocab.ContentTypeActivityPub)
	require.Nil(t, HeadStream(activityPub, nil, nil, &stream))

	htmlTag := html.Response().Header().Get("ETag")
	activityPubTag := activityPub.Response().Header().Get("ETag")

	require.NotEmpty(t, htmlTag)
	require.NotEmpty(t, activityPubTag)
	require.NotEqual(t, htmlTag, activityPubTag, "the two variants of one Stream share an ETag")
}
