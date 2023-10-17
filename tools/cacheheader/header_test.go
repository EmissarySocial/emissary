package cacheheader

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

/******************************************
* Single Directive Tests
******************************************/

func TestEmpty(t *testing.T) {
	header := ParseString("")
	require.True(t, header.IsNil())
}

func TestMaxAge(t *testing.T) {
	header := ParseString("max-age=1234")
	require.True(t, header.NotNil())
	require.Equal(t, int64(1234), header.MaxAge)

	s := header.String()
	require.Equal(t, "max-age=1234", s)
}

func TestMaxAge_Fail(t *testing.T) {
	header := ParseString("max-age=abc")
	require.True(t, header.IsNil())
	require.Equal(t, int64(0), header.MaxAge)

	s := header.String()
	require.Equal(t, "", s)
}

func TestSMaxAge(t *testing.T) {
	header := ParseString("s-maxage=1234")
	require.True(t, header.NotNil())
	require.Equal(t, int64(1234), header.SMaxAge)

	s := header.String()
	require.Equal(t, "s-maxage=1234", s)
}

func TestSMaxAge_Fail(t *testing.T) {
	header := ParseString("s-maxage=abc")
	require.True(t, header.IsNil())
	require.Equal(t, int64(0), header.SMaxAge)

	s := header.String()
	require.Equal(t, "", s)
}

func TestNoCache(t *testing.T) {
	header := ParseString("no-cache")
	require.True(t, header.NotNil())
	require.True(t, header.NoCache)

	s := header.String()
	require.Equal(t, "no-cache", s)
}

func TestNoStore(t *testing.T) {
	header := ParseString("no-store")
	require.True(t, header.NotNil())
	require.True(t, header.NoStore)

	s := header.String()
	require.Equal(t, "no-store", s)
}

func TestNoTransform(t *testing.T) {
	header := ParseString("no-transform")
	require.True(t, header.NotNil())
	require.True(t, header.NoTransform)

	s := header.String()
	require.Equal(t, "no-transform", s)
}

func TestMustRevalidate(t *testing.T) {
	header := ParseString("must-revalidate")
	require.True(t, header.NotNil())
	require.True(t, header.MustRevalidate)

	s := header.String()
	require.Equal(t, "must-revalidate", s)
}

func TestProxyRevalidate(t *testing.T) {
	header := ParseString("proxy-revalidate")
	require.True(t, header.NotNil())
	require.True(t, header.ProxyRevalidate)

	s := header.String()
	require.Equal(t, "proxy-revalidate", s)
}

func TestMustUnderstand(t *testing.T) {
	header := ParseString("must-understand")
	require.True(t, header.NotNil())
	require.True(t, header.MustUnderstand)

	s := header.String()
	require.Equal(t, "must-understand", s)
}

func TestPrivate(t *testing.T) {
	header := ParseString("private")
	require.True(t, header.NotNil())
	require.True(t, header.Private)

	s := header.String()
	require.Equal(t, "private", s)
}

func TestPublic(t *testing.T) {
	header := ParseString("public")
	require.True(t, header.NotNil())
	require.True(t, header.Public)

	s := header.String()
	require.Equal(t, "public", s)
}

func TestImmutable(t *testing.T) {
	header := ParseString("immutable")
	require.True(t, header.NotNil())
	require.True(t, header.Immutable)

	s := header.String()
	require.Equal(t, "immutable", s)
}

func TestStaleWhileRevalidate(t *testing.T) {
	header := ParseString("stale-while-revalidate=1234")
	require.True(t, header.NotNil())
	require.Equal(t, int64(1234), header.StaleWhileRevalidate)

	s := header.String()
	require.Equal(t, "stale-while-revalidate=1234", s)
}

func TestStaleWhileRevalidate_Fail(t *testing.T) {
	header := ParseString("stale-while-revalidate=abc")
	require.True(t, header.IsNil())
	require.Equal(t, int64(0), header.StaleWhileRevalidate)

	s := header.String()
	require.Equal(t, "", s)
}

func TestStaleIfError(t *testing.T) {
	header := ParseString("stale-if-error=1234")
	require.True(t, header.NotNil())
	require.Equal(t, int64(1234), header.StaleIfError)

	s := header.String()
	require.Equal(t, "stale-if-error=1234", s)
}

func TestStaleIfError_Fail(t *testing.T) {
	header := ParseString("stale-if-error=abc")
	require.True(t, header.IsNil())
	require.Equal(t, int64(0), header.StaleIfError)

	s := header.String()
	require.Equal(t, "", s)
}

func TestUnrecognized(t *testing.T) {
	header := ParseString("unrecognized")
	require.True(t, header.IsNil())

	s := header.String()
	require.Equal(t, "", s)
}

/******************************************
* Multiple Directive Tests
******************************************/

func TestMultiple(t *testing.T) {
	header := ParseString("public, max-age=604800, immutable")
	require.True(t, header.NotNil())
	require.True(t, header.Public)
	require.Equal(t, int64(604800), header.MaxAge)
	require.True(t, header.Immutable)

	s := header.String()
	require.Equal(t, "max-age=604800, public, immutable", s)
}

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
