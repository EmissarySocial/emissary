package build

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/benpate/data"
	"github.com/stretchr/testify/require"
)

// StepViewHTML publishes ETag/Last-Modified, which a browser will read as license to invent its own
// freshness lifetime unless the response also states a cache policy.  These tests pin that policy.

// stubViewHTMLBuilder is a build.Builder exposing only what StepViewHTML.execute reaches.  object()
// returns nil so the validator branch is skipped -- these tests are about Cache-Control alone.
type stubViewHTMLBuilder struct {
	Builder
	recorder *httptest.ResponseRecorder
}

// response implements the Builder interface, returning this stub's response recorder
func (b stubViewHTMLBuilder) response() http.ResponseWriter { return b.recorder }

// actionID implements the Builder interface, returning a fixed "view" action
func (b stubViewHTMLBuilder) actionID() string { return "view" }

// object implements the Builder interface. The stub owns no object, so the validator branch is skipped.
func (b stubViewHTMLBuilder) object() data.Object { return nil }

// execute implements the Builder interface. The stub renders nothing.
func (b stubViewHTMLBuilder) execute(_ io.Writer, _ string, _ any) error { return nil }

// runViewHTML executes the Step against a stub builder and returns the headers it wrote.
func runViewHTML(t *testing.T, step StepViewHTML) http.Header {

	t.Helper()

	builder := stubViewHTMLBuilder{recorder: httptest.NewRecorder()}
	step.execute(builder, io.Discard)

	return builder.recorder.Header()
}

// TestStepViewHTML_CacheControl covers the header that decides whether a browser may reuse this
// page without asking us first.
func TestStepViewHTML_CacheControl(t *testing.T) {

	t.Run("default denies caching", func(t *testing.T) {

		// A Template that says nothing must not get the permissive answer.  Most actions using
		// `view-html` are transactional (forms, modals, wizards), so silence has to fail closed.
		result := runViewHTML(t, StepViewHTML{Method: "get"})
		require.Equal(t, headers.DefaultCacheControlHTML, result.Get("Cache-Control"))
	})

	t.Run("template overrides", func(t *testing.T) {

		result := runViewHTML(t, StepViewHTML{Method: "get", CacheControl: "public, max-age=300"})
		require.Equal(t, "public, max-age=300", result.Get("Cache-Control"))
	})

	t.Run("Vary survives", func(t *testing.T) {

		// Cache-Control is written beside Vary, and a cache that honors one but not the other would
		// serve a peer's AS2 document to a browser.
		result := runViewHTML(t, StepViewHTML{Method: "get"})
		require.Equal(t, headers.VaryHTML, result.Get("Vary"))
	})
}
