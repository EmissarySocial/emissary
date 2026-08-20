package cacheheader

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

/******************************************
* Single Directive Tests
******************************************/

// TestEmpty verifies that an empty Cache-Control value parses into a nil Header
func TestEmpty(t *testing.T) {
	header := ParseString("")
	require.True(t, header.IsNil())
}

// TestMaxAge verifies that "max-age" round-trips through parse and String
func TestMaxAge(t *testing.T) {
	header := ParseString("max-age=1234")
	require.True(t, header.NotNil())
	require.Equal(t, int64(1234), header.MaxAge)

	s := header.String()
	require.Equal(t, "max-age=1234", s)
}

// TestMaxAge_Fail verifies that a non-numeric "max-age" is discarded
func TestMaxAge_Fail(t *testing.T) {
	header := ParseString("max-age=abc")
	require.True(t, header.IsNil())
	require.Equal(t, int64(0), header.MaxAge)

	s := header.String()
	require.Equal(t, "", s)
}

// TestSMaxAge verifies that "s-maxage" round-trips through parse and String
func TestSMaxAge(t *testing.T) {
	header := ParseString("s-maxage=1234")
	require.True(t, header.NotNil())
	require.Equal(t, int64(1234), header.SMaxAge)

	s := header.String()
	require.Equal(t, "s-maxage=1234", s)
}

// TestSMaxAge_Fail verifies that a non-numeric "s-maxage" is discarded
func TestSMaxAge_Fail(t *testing.T) {
	header := ParseString("s-maxage=abc")
	require.True(t, header.IsNil())
	require.Equal(t, int64(0), header.SMaxAge)

	s := header.String()
	require.Equal(t, "", s)
}

// TestNoCache verifies that "no-cache" round-trips through parse and String
func TestNoCache(t *testing.T) {
	header := ParseString("no-cache")
	require.True(t, header.NotNil())
	require.True(t, header.NoCache)

	s := header.String()
	require.Equal(t, "no-cache", s)
}

// TestNoStore verifies that "no-store" round-trips through parse and String
func TestNoStore(t *testing.T) {
	header := ParseString("no-store")
	require.True(t, header.NotNil())
	require.True(t, header.NoStore)

	s := header.String()
	require.Equal(t, "no-store", s)
}

// TestNoTransform verifies that "no-transform" round-trips through parse and String
func TestNoTransform(t *testing.T) {
	header := ParseString("no-transform")
	require.True(t, header.NotNil())
	require.True(t, header.NoTransform)

	s := header.String()
	require.Equal(t, "no-transform", s)
}

// TestMustRevalidate verifies that "must-revalidate" round-trips through parse and String
func TestMustRevalidate(t *testing.T) {
	header := ParseString("must-revalidate")
	require.True(t, header.NotNil())
	require.True(t, header.MustRevalidate)

	s := header.String()
	require.Equal(t, "must-revalidate", s)
}

// TestProxyRevalidate verifies that "proxy-revalidate" round-trips through parse and String
func TestProxyRevalidate(t *testing.T) {
	header := ParseString("proxy-revalidate")
	require.True(t, header.NotNil())
	require.True(t, header.ProxyRevalidate)

	s := header.String()
	require.Equal(t, "proxy-revalidate", s)
}

// TestMustUnderstand verifies that "must-understand" round-trips through parse and String
func TestMustUnderstand(t *testing.T) {
	header := ParseString("must-understand")
	require.True(t, header.NotNil())
	require.True(t, header.MustUnderstand)

	s := header.String()
	require.Equal(t, "must-understand", s)
}

// TestPrivate verifies that "private" round-trips through parse and String
func TestPrivate(t *testing.T) {
	header := ParseString("private")
	require.True(t, header.NotNil())
	require.True(t, header.Private)

	s := header.String()
	require.Equal(t, "private", s)
}

// TestPublic verifies that "public" round-trips through parse and String
func TestPublic(t *testing.T) {
	header := ParseString("public")
	require.True(t, header.NotNil())
	require.True(t, header.Public)

	s := header.String()
	require.Equal(t, "public", s)
}

// TestImmutable verifies that "immutable" round-trips through parse and String
func TestImmutable(t *testing.T) {
	header := ParseString("immutable")
	require.True(t, header.NotNil())
	require.True(t, header.Immutable)

	s := header.String()
	require.Equal(t, "immutable", s)
}

// TestStaleWhileRevalidate verifies that "stale-while-revalidate" round-trips through parse and String
func TestStaleWhileRevalidate(t *testing.T) {
	header := ParseString("stale-while-revalidate=1234")
	require.True(t, header.NotNil())
	require.Equal(t, int64(1234), header.StaleWhileRevalidate)

	s := header.String()
	require.Equal(t, "stale-while-revalidate=1234", s)
}

// TestStaleWhileRevalidate_Fail verifies that a non-numeric "stale-while-revalidate" is discarded
func TestStaleWhileRevalidate_Fail(t *testing.T) {
	header := ParseString("stale-while-revalidate=abc")
	require.True(t, header.IsNil())
	require.Equal(t, int64(0), header.StaleWhileRevalidate)

	s := header.String()
	require.Equal(t, "", s)
}

// TestStaleIfError verifies that "stale-if-error" round-trips through parse and String
func TestStaleIfError(t *testing.T) {
	header := ParseString("stale-if-error=1234")
	require.True(t, header.NotNil())
	require.Equal(t, int64(1234), header.StaleIfError)

	s := header.String()
	require.Equal(t, "stale-if-error=1234", s)
}

// TestStaleIfError_Fail verifies that a non-numeric "stale-if-error" is discarded
func TestStaleIfError_Fail(t *testing.T) {
	header := ParseString("stale-if-error=abc")
	require.True(t, header.IsNil())
	require.Equal(t, int64(0), header.StaleIfError)

	s := header.String()
	require.Equal(t, "", s)
}

// TestUnrecognized verifies that an unknown directive is discarded instead of echoed back
func TestUnrecognized(t *testing.T) {
	header := ParseString("unrecognized")
	require.True(t, header.IsNil())

	s := header.String()
	require.Equal(t, "", s)
}

/******************************************
* Multiple Directive Tests
******************************************/

// TestMultiple verifies that several directives parse together and re-serialize in canonical order
func TestMultiple(t *testing.T) {
	header := ParseString("public, max-age=604800, immutable")
	require.True(t, header.NotNil())
	require.True(t, header.Public)
	require.Equal(t, int64(604800), header.MaxAge)
	require.True(t, header.Immutable)

	s := header.String()
	require.Equal(t, "max-age=604800, public, immutable", s)
}

// TestParse_SingleValue verifies parsing an http.Header whose Cache-Control is one combined string
func TestParse_SingleValue(t *testing.T) {
	header := http.Header{
		"Cache-Control": []string{"public, max-age=604800, immutable"},
	}

	result := Parse(header)
	require.True(t, result.NotNil())
	require.True(t, result.Public)
	require.Equal(t, int64(604800), result.MaxAge)
	require.True(t, result.Immutable)

	s := result.String()
	require.Equal(t, "max-age=604800, public, immutable", s)
}

// TestParse_MultiValue verifies parsing an http.Header whose Cache-Control is split across several values
func TestParse_MultiValue(t *testing.T) {
	header := http.Header{
		"Cache-Control": []string{"public", "max-age=604800", "immutable"},
	}

	result := Parse(header)
	require.True(t, result.NotNil())
	require.True(t, result.Public)
	require.Equal(t, int64(604800), result.MaxAge)
	require.True(t, result.Immutable)

	s := result.String()
	require.Equal(t, "max-age=604800, public, immutable", s)
}
