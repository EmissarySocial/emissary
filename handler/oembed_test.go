package handler

import (
	"encoding/json"
	"encoding/xml"
	"net"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/oembed"
	"github.com/benpate/rosetta/lenient"
	"github.com/stretchr/testify/require"
)

// TestOEmbedResponse_XML pins the XML document that this domain serves.  It is the
// regression test for the bug that made every "format=xml" request fail: the document
// used to be a map, and encoding/xml cannot marshal a map at all.
func TestOEmbedResponse_XML(t *testing.T) {

	response := oembed.NewLink("Hello World")
	response.CacheAge = oEmbedCacheAge
	response.ProviderName = "Bandwagon"
	response.ProviderURL = "https://bandwagon.fm"

	result, err := xml.Marshal(response)

	require.Nil(t, err)

	// The root element must be <oembed>, per the spec
	require.True(t, strings.HasPrefix(string(result), "<oembed>"))
	require.True(t, strings.HasSuffix(string(result), "</oembed>"))

	require.Contains(t, string(result), "<version>1.0</version>")
	require.Contains(t, string(result), "<type>link</type>")
	require.Contains(t, string(result), "<title>Hello World</title>")
	require.Contains(t, string(result), "<cache_age>86400</cache_age>")
	require.Contains(t, string(result), "<provider_name>Bandwagon</provider_name>")
	require.Contains(t, string(result), "<provider_url>https://bandwagon.fm</provider_url>")

	// Absent thumbnails must not produce empty elements
	require.NotContains(t, string(result), "thumbnail")
}

// TestOEmbedResponse_XML_Thumbnail proves the optional thumbnail fields marshal when present
func TestOEmbedResponse_XML_Thumbnail(t *testing.T) {

	response := oembed.NewLink("")
	response.SetThumbnail("https://bandwagon.fm/icon.webp", oEmbedThumbnailSize, oEmbedThumbnailSize)

	result, err := xml.Marshal(response)

	require.Nil(t, err)
	require.Contains(t, string(result), "<thumbnail_url>https://bandwagon.fm/icon.webp</thumbnail_url>")
	require.Contains(t, string(result), "<thumbnail_height>300</thumbnail_height>")
	require.Contains(t, string(result), "<thumbnail_width>300</thumbnail_width>")
}

// TestOEmbedResponse_JSON pins the JSON field names that oEmbed consumers read
func TestOEmbedResponse_JSON(t *testing.T) {

	response := oembed.NewLink("Hello World")
	response.CacheAge = oEmbedCacheAge
	response.ProviderName = "Bandwagon"
	response.ProviderURL = "https://bandwagon.fm"

	result, err := json.Marshal(response)

	require.Nil(t, err)

	unmarshalled := make(map[string]any)
	require.Nil(t, json.Unmarshal(result, &unmarshalled))

	require.Equal(t, "1.0", unmarshalled["version"])
	require.Equal(t, "link", unmarshalled["type"])
	require.Equal(t, "Hello World", unmarshalled["title"])
	require.Equal(t, float64(86400), unmarshalled["cache_age"])
	require.Equal(t, "Bandwagon", unmarshalled["provider_name"])
	require.Equal(t, "https://bandwagon.fm", unmarshalled["provider_url"])

	// XMLName must never leak into the JSON document
	require.NotContains(t, unmarshalled, "XMLName")

	// Optional fields are omitted entirely, not sent as empty values
	require.NotContains(t, unmarshalled, "thumbnail_url")
	require.NotContains(t, unmarshalled, "thumbnail_height")
	require.NotContains(t, unmarshalled, "thumbnail_width")
}

// TestOEmbedResponse_Write proves that the documents this domain serves reach the wire
// through the library's writer: validated, correctly encoded, and correctly typed.
func TestOEmbedResponse_Write(t *testing.T) {

	response := oembed.NewLink("Hello World")
	response.CacheAge = oEmbedCacheAge
	setOEmbedThumbnail(&response, "bandwagon.fm", "https://bandwagon.fm/@user/attachments/123")

	// An empty format means "no preference", and is answered with JSON
	for _, format := range []string{"", oembed.FormatJSON} {

		recorder := httptest.NewRecorder()
		require.Nil(t, oembed.WriteResponse(recorder, response, format))

		require.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
		require.Contains(t, recorder.Body.String(), `"thumbnail_width":300`)
	}

	// XML is served as text/xml, rooted at <oembed>
	{
		recorder := httptest.NewRecorder()
		require.Nil(t, oembed.WriteResponse(recorder, response, oembed.FormatXML))

		require.Equal(t, "text/xml; charset=utf-8", recorder.Header().Get("Content-Type"))
		require.Contains(t, recorder.Body.String(), "<oembed>")
		require.Contains(t, recorder.Body.String(), "<thumbnail_width>300</thumbnail_width>")
	}

	// RULE: an unsupported format is a 501, and writes nothing at all
	{
		recorder := httptest.NewRecorder()
		err := oembed.WriteResponse(recorder, response, "yaml")

		require.NotNil(t, err)
		require.Equal(t, 501, derp.ErrorCode(err))
		require.Zero(t, recorder.Body.Len())
	}
}

// TestNormalizeHostname proves that a hostname is reduced to a comparable value,
// whatever shape the consumer sent it in.
func TestNormalizeHostname(t *testing.T) {

	test := func(value string, expected string) {
		t.Helper()
		require.Equal(t, expected, normalizeHostname(value), "normalizing %q", value)
	}

	// Bare hostnames
	test("bandwagon.fm", "bandwagon.fm")
	test("BANDWAGON.FM", "bandwagon.fm")
	test("www.bandwagon.fm", "bandwagon.fm")

	// Full URLs
	test("https://bandwagon.fm/@user", "bandwagon.fm")
	test("http://bandwagon.fm/@user", "bandwagon.fm")
	test("HTTPS://BandWagon.FM/@user", "bandwagon.fm")
	test("https://www.bandwagon.fm/@user", "bandwagon.fm")

	// Ports are never part of the comparison
	test("bandwagon.fm:8080", "bandwagon.fm")
	test("https://bandwagon.fm:8080/@user", "bandwagon.fm")
	test("localhost:8080", "localhost")

	// Degenerate inputs must not panic
	test("", "")
	test("/", "")
	test("//", "")
	test(":", "")
	test("https://", "")
}

// TestNotSameHostname is the regression test for the reported bug: the oEmbed
// endpoint 404'd on URLs that the request router happily accepts.
func TestNotSameHostname(t *testing.T) {

	const hostname = "bandwagon.fm"

	accepted := func(value string) {
		t.Helper()
		require.False(t, notSameHostname(value, hostname), "%q should belong to %q", value, hostname)
	}

	rejected := func(value string) {
		t.Helper()
		require.True(t, notSameHostname(value, hostname), "%q should NOT belong to %q", value, hostname)
	}

	// The URL from the original bug report
	accepted("https://bandwagon.fm/@681c4330b949b6b581af8a51")

	// ...and the variants that used to 404
	accepted("https://bandwagon.fm:8080/@681c4330b949b6b581af8a51")
	accepted("https://www.bandwagon.fm/@681c4330b949b6b581af8a51")
	accepted("https://BandWagon.fm/@681c4330b949b6b581af8a51")
	accepted("bandwagon.fm/@681c4330b949b6b581af8a51")
	accepted("bandwagon.fm")

	// Other domains are still refused
	rejected("https://example.com/@681c4330b949b6b581af8a51")
	rejected("https://bandwagon.fm.evil.com/@681c4330b949b6b581af8a51")
	rejected("https://evil.com/?bandwagon.fm")
	rejected("")
}

// TestOEmbedToken proves that every URL which renders a record resolves to the
// same record token -- including the trailing slashes and action names that
// used to produce a lookup for a token that cannot exist.
func TestOEmbedToken(t *testing.T) {

	test := func(path string, expected string) {
		t.Helper()
		require.Equal(t, expected, oEmbedToken(path), "tokenizing %q", path)
	}

	// The domain home page
	test("", "")
	test("/", "")

	// User profiles
	test("/@681c4330b949b6b581af8a51", "@681c4330b949b6b581af8a51")
	test("/@681c4330b949b6b581af8a51/", "@681c4330b949b6b581af8a51")
	test("/@681c4330b949b6b581af8a51/pub", "@681c4330b949b6b581af8a51")
	test("/@benpate/followers", "@benpate")

	// Streams
	test("/my-stream", "my-stream")
	test("/my-stream/", "my-stream")
	test("/my-stream/edit", "my-stream")

	// Paths arrive from url.Parse, so they are already rooted -- but a bare
	// value must not lose its first character either.
	test("my-stream", "my-stream")
}

// TestIsLocalMediaURL proves that only URLs this domain can actually resize are decorated
func TestIsLocalMediaURL(t *testing.T) {

	const hostname = "bandwagon.fm"

	local := func(value string) {
		t.Helper()
		require.True(t, isLocalMediaURL(value, hostname), "%q should be local", value)
	}

	remote := func(value string) {
		t.Helper()
		require.False(t, isLocalMediaURL(value, hostname), "%q should NOT be local", value)
	}

	// URLs this domain serves
	local("https://bandwagon.fm/@user/attachments/123")
	local("https://www.bandwagon.fm/@user/attachments/123")
	local("https://bandwagon.fm:8080/@user/attachments/123")
	local("/@user/attachments/123")

	// URLs some other server serves
	remote("https://mastodon.social/media/abc.jpg")
	remote("https://cdn.example.com/logo.png")

	// A URL that already has a query string cannot take a resize query
	remote("https://bandwagon.fm/@user/attachments/123?v=2")
	remote("/@user/attachments/123?v=2")
}

// TestOEmbedResponse_setThumbnail proves local icons are resized and remote icons are left alone
func TestOEmbedResponse_setThumbnail(t *testing.T) {

	const hostname = "bandwagon.fm"

	// An empty icon produces no thumbnail at all
	{
		response := oembed.Response{}
		setOEmbedThumbnail(&response, hostname, "")

		require.Zero(t, response.ThumbnailURL)
		require.Zero(t, response.ThumbnailHeight)
		require.Zero(t, response.ThumbnailWidth)
	}

	// A local icon is handed to this domain's mediaserver
	{
		response := oembed.Response{}
		setOEmbedThumbnail(&response, hostname, "https://bandwagon.fm/@user/attachments/123")

		require.Equal(t, "https://bandwagon.fm/@user/attachments/123.webp?height=300&width=300", response.ThumbnailURL)
		require.Equal(t, lenient.Int64(oEmbedThumbnailSize), response.ThumbnailHeight)
		require.Equal(t, lenient.Int64(oEmbedThumbnailSize), response.ThumbnailWidth)
	}

	// A remote icon is published untouched -- its server cannot answer a resize request
	{
		response := oembed.Response{}
		setOEmbedThumbnail(&response, hostname, "https://mastodon.social/media/abc.jpg")

		require.Equal(t, "https://mastodon.social/media/abc.jpg", response.ThumbnailURL)
		require.Equal(t, lenient.Int64(oEmbedThumbnailSize), response.ThumbnailHeight)
		require.Equal(t, lenient.Int64(oEmbedThumbnailSize), response.ThumbnailWidth)
	}

	// The decorated URL must remain parseable, with exactly one query string
	{
		response := oembed.Response{}
		setOEmbedThumbnail(&response, hostname, "/@user/attachments/123")

		parsed, err := url.Parse(response.ThumbnailURL)

		require.Nil(t, err)
		require.Equal(t, "/@user/attachments/123.webp", parsed.Path)
		require.Equal(t, "300", parsed.Query().Get("height"))
		require.Equal(t, "300", parsed.Query().Get("width"))
	}
}

// FuzzNormalizeHostname proves that arbitrary consumer input cannot panic the
// domain guard, and that its result is always safe to compare.
func FuzzNormalizeHostname(f *testing.F) {

	f.Add("bandwagon.fm")
	f.Add("https://bandwagon.fm:8080/@user")
	f.Add("https://www.BANDWAGON.fm/@user?x=1#y")
	f.Add("")
	f.Add("//")
	f.Add("::1")
	f.Add("http://[::1]:8080/")
	f.Add("\x00\xff")

	f.Fuzz(func(t *testing.T, value string) {

		result := normalizeHostname(value)

		// A normalized hostname never carries a port, a path, or a scheme
		require.NotContains(t, result, "/")

		// RULE: An unbracketed IPv6 literal ("::1") legitimately contains colons --
		// uri.NormalizeHost documents that it returns IPv6 in exactly that canonical
		// form. Only a colon in a NON-IP result would be a port that survived.
		if net.ParseIP(result) == nil {
			require.NotContains(t, result, ":")
		}

		// Normalizing is idempotent, so comparing two normalized values is stable
		require.Equal(t, result, normalizeHostname(result))
	})
}

// FuzzOEmbedToken proves that no URL path can panic the record lookup, and that
// the token is always a single path segment.
func FuzzOEmbedToken(f *testing.F) {

	f.Add("/@681c4330b949b6b581af8a51")
	f.Add("/@681c4330b949b6b581af8a51/pub")
	f.Add("/my-stream/")
	f.Add("")
	f.Add("/")
	f.Add("//////")
	f.Add("\x00\xff")

	f.Fuzz(func(t *testing.T, path string) {

		token := oEmbedToken(path)

		// A token is always a single segment
		require.NotContains(t, token, "/")

		// Tokenizing is idempotent
		require.Equal(t, token, oEmbedToken(token))
	})
}
