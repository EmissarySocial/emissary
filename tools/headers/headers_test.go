package headers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/vocab"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// testObject is a stand-in for any model embedding journal.Journal.
type testObject struct {
	etag    string
	updated int64
}

func (o testObject) ETag() string   { return o.etag }
func (o testObject) Updated() int64 { return o.updated }

// TestVariantOf covers the mapping from an "Accept" header to a Variant.
func TestVariantOf(t *testing.T) {

	// The full Accept-parsing table lives in hannibal, which owns that reading. These rows exist to
	// prove the two answers are wired together, not to re-test the parser.
	table := []struct {
		name   string
		accept string
		expect Variant
	}{
		{"absent header", "", VariantHTML},
		{"curl default", "*/*", VariantHTML},
		{"browser", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", VariantHTML},
		{"mastodon", `application/activity+json, application/ld+json; profile="https://www.w3.org/ns/activitystreams"`, VariantActivityPub},
		{"activity+json alone", vocab.ContentTypeActivityPub, VariantActivityPub},
		{"q-values reorder the list", "text/html;q=0.9, application/activity+json;q=1.0", VariantActivityPub},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {

			request := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)

			if test.accept != "" {
				request.Header.Set("Accept", test.accept)
			}

			require.Equal(t, test.expect, VariantOf(request), "Accept: %q", test.accept)
		})
	}
}

// TestVariantHeaders covers the exact Content-Type and Vary strings that both verbs emit.
func TestVariantHeaders(t *testing.T) {

	require.Equal(t, vocab.ContentTypeActivityPub, ContentType(VariantActivityPub))
	require.Equal(t, "text/html; charset=UTF-8", ContentType(VariantHTML))

	require.Equal(t, "Accept", Vary(VariantActivityPub))
	require.Equal(t, "Cookie, HX-Request, Accept", Vary(VariantHTML))

	header := http.Header{}
	SetVariant(header, VariantActivityPub)
	require.Equal(t, vocab.ContentTypeActivityPub, header.Get("Content-Type"))
	require.Equal(t, "Accept", header.Get("Vary"))

	header = http.Header{}
	SetVariant(header, VariantHTML)
	require.Equal(t, "text/html; charset=UTF-8", header.Get("Content-Type"))
	require.Equal(t, "Cookie, HX-Request, Accept", header.Get("Vary"))
}

// TestETag covers the two properties that make an entity-tag correct: it is well-formed, and it
// identifies one *representation* rather than one object.
func TestETag(t *testing.T) {

	object := testObject{etag: "42", updated: 1754913127000}

	html := ETag(VariantHTML, object)
	activityPub := ETag(VariantActivityPub, object)

	t.Run("is a well-formed weak tag", func(t *testing.T) {
		// RFC 9110 section 8.8.3 requires DQUOTE-delimited tags; an unquoted value is malformed.
		// The "W/" prefix marks these as weak, which is what a revision-derived tag can honestly
		// promise -- see projects/DEPLOY-IDENTIFIER.md.
		for _, tag := range []string{html, activityPub} {
			require.True(t, strings.HasPrefix(tag, `W/"`), "tag is not a weak tag: %q", tag)
			require.True(t, strings.HasSuffix(tag, `"`), "tag does not end with a quote: %q", tag)
			require.NotContains(t, tag[3:len(tag)-1], `"`, "tag body contains a quote: %q", tag)
		}
	})

	t.Run("differs between variants of one object", func(t *testing.T) {
		// This is the whole point: sharing a tag across representations would let a client holding
		// the HTML tag be answered 304 when it asks for ActivityStreams.
		require.NotEqual(t, html, activityPub)
	})

	t.Run("changes when the object changes", func(t *testing.T) {
		require.NotEqual(t, html, ETag(VariantHTML, testObject{etag: "43", updated: 1754913127000}))
	})

	t.Run("is stable for the same object and variant", func(t *testing.T) {
		require.Equal(t, html, ETag(VariantHTML, object))
	})
}

// TestEntityTag covers tag assembly for callers outside the two-variant scheme, such as CSS views.
func TestEntityTag(t *testing.T) {

	tag := EntityTag("42", "css")

	require.True(t, strings.HasPrefix(tag, `W/"`))
	require.True(t, strings.HasSuffix(tag, `"`))
	require.Contains(t, tag, "42")
	require.Contains(t, tag, "css")
	require.NotEqual(t, tag, EntityTag("42", "html"))

	// A Validator is an interface, so a value carrying a quote or a control character would
	// otherwise terminate the header early.
	require.Equal(t, `W/"abc"`, EntityTag(`a"b`+"\x7Fc"))
}

// TestLastModified covers the one date format an HTTP header may use.
func TestLastModified(t *testing.T) {

	object := testObject{etag: "42", updated: 1754913127000}
	value := LastModified(object)

	// It must parse as an IMF-fixdate and must NOT parse as RFC3339 -- the two have been swapped
	// in this codebase before, in both directions.
	parsed, err := time.Parse(http.TimeFormat, value)
	require.Nil(t, err, "Last-Modified is not an HTTP-date: %q", value)
	require.Equal(t, int64(1754913127000), parsed.UnixMilli())

	_, err = time.Parse(time.RFC3339, value)
	require.NotNil(t, err, "Last-Modified parsed as RFC3339, so it is the wrong format: %q", value)

	// http.TimeFormat ends in a literal "GMT", so a non-UTC value would be labelled wrongly.
	require.Contains(t, value, "GMT")
}

// TestDefaultCacheControlHTML covers the one property that makes the default safe: a page carrying
// it can never be reused without asking the origin first.
func TestDefaultCacheControlHTML(t *testing.T) {

	// `no-cache` is the load-bearing directive -- it permits storage but forbids reuse without
	// revalidation, which is exactly what denies a browser its heuristic freshness calculation.
	directives := cacheheader.ParseString(DefaultCacheControlHTML)

	require.True(t, directives.NoCache, "the default must forbid reuse without revalidation")
	require.True(t, directives.Private, "a Cookie-varying page must not land in a shared cache")
	require.True(t, directives.NotCacheAllowed())
}

// TestSetValidators covers both validators reaching the response together.
func TestSetValidators(t *testing.T) {

	object := testObject{etag: "42", updated: 1754913127000}

	header := http.Header{}
	SetValidators(header, VariantActivityPub, object)
	require.Equal(t, ETag(VariantActivityPub, object), header.Get("ETag"))
	require.Equal(t, LastModified(object), header.Get("Last-Modified"))

	header = http.Header{}
	SetAll(header, VariantActivityPub, object)
	require.Equal(t, vocab.ContentTypeActivityPub, header.Get("Content-Type"))
	require.Equal(t, "Accept", header.Get("Vary"))
	require.NotEmpty(t, header.Get("ETag"))
	require.NotEmpty(t, header.Get("Last-Modified"))
}

// TestContentTypeMatchesEcho covers ContentTypeHTML against what Echo actually writes, and that a
// Content-Type set before ctx.JSON survives.
func TestContentTypeMatchesEcho(t *testing.T) {

	// HEAD reports this string without rendering, so an Echo upgrade that respelled the charset
	// would otherwise put HEAD and GET back into disagreement, silently.
	e := echo.New()

	t.Run("ctx.HTML matches ContentType(VariantHTML)", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), recorder)

		require.Nil(t, ctx.HTML(http.StatusOK, "<p>hello</p>"))
		require.Equal(t, ContentType(VariantHTML), recorder.Header().Get("Content-Type"))
	})

	t.Run("ctx.JSON preserves a pre-set Content-Type", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), recorder)

		SetVariant(ctx.Response().Header(), VariantActivityPub)
		require.Nil(t, ctx.JSON(http.StatusOK, map[string]string{"type": "Person"}))

		require.Equal(t, vocab.ContentTypeActivityPub, recorder.Header().Get("Content-Type"))
		require.Equal(t, "Accept", recorder.Header().Get("Vary"))
	})
}
