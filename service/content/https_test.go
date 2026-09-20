package content

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// newTestSource returns an adapter and a StreamSource pointed at a test server.
// RULE: allowPrivateIPs is TRUE because httptest listens on 127.0.0.1, which the SSRF guard
// refuses outright -- a test built with FALSE would pass against a broken adapter.
func newTestSource(t *testing.T, handler http.HandlerFunc) (HTTPS, model.StreamSource) {

	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	source := model.NewStreamSource()
	source.Method = model.StreamSourceMethodHTTPS
	source.URL = server.URL + "/docs/page.md"

	return NewHTTPS(true), source
}

// markdownHandler serves a body as text/plain, which is what every forge surveyed returns
func markdownHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

func TestHTTPS_Protocol(t *testing.T) {
	require.Equal(t, model.StreamSourceMethodHTTPS, NewHTTPS(false).Protocol())
}

func TestHTTPS_Subscribe(t *testing.T) {
	err := NewHTTPS(false).Subscribe(context.Background(), model.NewStreamSource(), "https://example.com/hook")
	require.True(t, derp.IsNotImplemented(err), "a file cannot offer a subscription")
}

// TestHTTPS_Version_NewValidator returns whatever validator the origin offered
func TestHTTPS_Version_NewValidator(t *testing.T) {

	adapter, source := newTestSource(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("# Hello"))
	})

	version, err := adapter.Version(context.Background(), source)
	require.NoError(t, err)
	require.Equal(t, `"abc123"`, version)
}

// TestHTTPS_Version_NotModified keeps the stored validator when the origin answers 304
func TestHTTPS_Version_NotModified(t *testing.T) {

	var sent string

	adapter, source := newTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		sent = r.Header.Get("If-None-Match")
		w.WriteHeader(http.StatusNotModified)
	})

	source.Version = `"stored"`

	version, err := adapter.Version(context.Background(), source)
	require.NoError(t, err)
	require.Equal(t, `"stored"`, version, "a 304 leaves the version alone")
	require.Equal(t, `"stored"`, sent, "the stored validator is what makes the request conditional")
}

// TestHTTPS_Version_NoValidator is not an error -- cgit and SourceHut offer none
func TestHTTPS_Version_NoValidator(t *testing.T) {

	adapter, source := newTestSource(t, markdownHandler("# Hello"))

	version, err := adapter.Version(context.Background(), source)
	require.NoError(t, err)
	require.Empty(t, version, "a source with no validator simply fetches every time")
}

// TestHTTPS_Version_SendsNoConditionalHeaderWhenUnset confirms a first sync asks unconditionally
func TestHTTPS_Version_SendsNoConditionalHeaderWhenUnset(t *testing.T) {

	_, hasHeader := false, false

	adapter, source := newTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		_, hasHeader = r.Header["If-None-Match"]
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("# Hello"))
	})

	_, err := adapter.Version(context.Background(), source)
	require.NoError(t, err)
	require.False(t, hasHeader, "a record with no stored version sends no conditional header")
}

// TestHTTPS_Fetch splits front matter off a Markdown file and reports the declared format
func TestHTTPS_Fetch(t *testing.T) {

	adapter, source := newTestSource(t, markdownHandler("---\ntitle: Getting Started\n---\n# Body\n"))

	item, err := adapter.Fetch(context.Background(), source, "")
	require.NoError(t, err)

	require.Equal(t, model.ContentFormatMarkdown, item.Format)
	require.Equal(t, "# Body\n", string(item.Source))
	require.Equal(t, "Getting Started", item.Meta.GetString("title"))
	require.NotEmpty(t, item.Hash)
}

// TestHTTPS_Fetch_MediaTypes pins which media types this version accepts
func TestHTTPS_Fetch_MediaTypes(t *testing.T) {

	test := func(name string, contentType string, expectError bool) {
		t.Run(name, func(t *testing.T) {

			adapter, source := newTestSource(t, func(w http.ResponseWriter, _ *http.Request) {
				if contentType != "" {
					w.Header().Set("Content-Type", contentType)
				} else {
					// Go adds a Content-Type by sniffing unless it is explicitly blanked
					w.Header()["Content-Type"] = nil
				}
				_, _ = w.Write([]byte("# Body\n"))
			})

			_, err := adapter.Fetch(context.Background(), source, "")

			if expectError {
				require.Error(t, err)
				require.True(t, derp.IsClientError(err), "an unusable source is the author's to fix")
				return
			}

			require.NoError(t, err)
		})
	}

	test("plain text", "text/plain; charset=utf-8", false)
	test("markdown", "text/markdown", false)
	test("markdown with charset", "text/markdown; charset=UTF-8", false)
	test("x-markdown", "text/x-markdown", false)
	test("uppercase", "TEXT/PLAIN", false)

	// RULE: This is the C2 guard.  A forge's file PAGE answers text/html and differs from the raw
	// URL by one path segment, so this is what stops a pasted browser URL from becoming a body.
	test("html page", "text/html; charset=utf-8", true)
	test("json", "application/json", true)
	test("octet stream", "application/octet-stream", true)
	test("unparseable", "text/plain; charset=", true)
	test("absent", "", true)
}

// TestHTTPS_Fetch_NoContentTypeSaysSo pins the MESSAGE, not just the refusal.  mime.ParseMediaType
// rejects an empty string on its own, so the explicit guard earns its place only by telling the
// author which of the two problems they have.
func TestHTTPS_Fetch_NoContentTypeSaysSo(t *testing.T) {

	adapter, source := newTestSource(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header()["Content-Type"] = nil
		_, _ = w.Write([]byte("# Body\n"))
	})

	_, err := adapter.Fetch(context.Background(), source, "")

	require.Error(t, err)
	require.Contains(t, derp.Message(derp.RootCause(err)), "did not declare")
}

// TestHTTPS_Fetch_SizeLimit keeps a body at the cap and refuses one past it
func TestHTTPS_Fetch_SizeLimit(t *testing.T) {

	t.Run("at the limit", func(t *testing.T) {
		adapter, source := newTestSource(t, markdownHandler(strings.Repeat("a", maxContentBytes)))
		item, err := adapter.Fetch(context.Background(), source, "")
		require.NoError(t, err)
		require.Len(t, item.Source, maxContentBytes)
	})

	t.Run("one byte past the limit", func(t *testing.T) {
		adapter, source := newTestSource(t, markdownHandler(strings.Repeat("a", maxContentBytes+1)))
		_, err := adapter.Fetch(context.Background(), source, "")
		require.Error(t, err)
		require.True(t, derp.IsClientError(err))
	})
}

// TestHTTPS_Fetch_StatusCodes sorts the author's problems from Emissary's
func TestHTTPS_Fetch_StatusCodes(t *testing.T) {

	test := func(name string, status int, wantClientError bool) {
		t.Run(name, func(t *testing.T) {

			adapter, source := newTestSource(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})

			_, err := adapter.Fetch(context.Background(), source, "")
			require.Error(t, err)
			require.Equal(t, wantClientError, derp.IsClientError(err), "status %d", status)
		})
	}

	test("not found", http.StatusNotFound, true)
	test("gone", http.StatusGone, true)
	test("unauthorized", http.StatusUnauthorized, true)
	test("forbidden", http.StatusForbidden, true)
	test("teapot", http.StatusTeapot, true)

	// RULE: A 5xx is Emissary's to retry, so it is filed as a defect rather than shown to the author
	test("server error", http.StatusInternalServerError, false)
	test("bad gateway", http.StatusBadGateway, false)
}

// TestHTTPS_Fetch_UnreachableHost reports a transport failure as Emissary's problem
func TestHTTPS_Fetch_UnreachableHost(t *testing.T) {

	source := model.NewStreamSource()
	source.URL = "http://127.0.0.1:1/never-listening.md"

	_, err := NewHTTPS(true).Fetch(context.Background(), source, "")
	require.Error(t, err)
	require.False(t, derp.IsClientError(err), "a network failure may succeed on a later attempt")
}
