package httpcache

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

/******************************************
 * Test helpers
 ******************************************/

// newTestCache returns an HTTPCache backed by an in-memory adapter, plus that adapter,
// so a test can inspect exactly which keys were written.
func newTestCache(options ...Option) (HTTPCache, testAdapter) {

	adapter := testAdapter{}
	cache := HTTPCache{Adapter: adapter}
	cache.With(options...)

	return cache, adapter
}

// newTestRequest returns a GET request for the provided URL, with optional header pairs.
func newTestRequest(t *testing.T, address string, headers ...string) *http.Request {

	t.Helper()

	request, err := http.NewRequest(http.MethodGet, address, nil)
	require.NoError(t, err)

	for index := 0; index+1 < len(headers); index += 2 {
		request.Header.Set(headers[index], headers[index+1])
	}

	return request
}

// newTestResponse returns a 200 response carrying the provided body, with optional header pairs.
func newTestResponse(body string, headers ...string) *http.Response {

	response := &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}

	for index := 0; index+1 < len(headers); index += 2 {
		response.Header.Set(headers[index], headers[index+1])
	}

	return response
}

/******************************************
 * Embedding contract
 ******************************************/

// TestHTTPCache_GetSetAreTheAdapters pins the fact that HTTPCache declares no Get or Set
// of its own: both are promoted from the embedded Adapter and reach storage directly.
func TestHTTPCache_GetSetAreTheAdapters(t *testing.T) {

	cache, adapter := newTestCache()

	cache.Set("key", "value", time.Second)
	require.Equal(t, "value", adapter["key"], "Set writes straight through to the Adapter")

	value, ok := cache.Get("key")
	require.True(t, ok)
	require.Equal(t, "value", value)

	cache.Delete("key")
	_, ok = cache.Get("key")
	require.False(t, ok)
}

/******************************************
 * Options
 ******************************************/

// TestWithTTL confirms the option sets the cache's default lifetime.
func TestWithTTL(t *testing.T) {

	cache, _ := newTestCache(WithTTL(10 * time.Minute))
	require.Equal(t, 10*time.Minute, cache.ttl)
}

// TestWith_AppliesEveryOption confirms options are applied in order, so the last one wins.
func TestWith_AppliesEveryOption(t *testing.T) {

	cache, _ := newTestCache(WithTTL(time.Minute), WithTTL(time.Hour))
	require.Equal(t, time.Hour, cache.ttl)
}

// TestWith_NoOptions confirms an empty option list leaves the zero TTL untouched.
func TestWith_NoOptions(t *testing.T) {

	cache, _ := newTestCache()
	require.Equal(t, time.Duration(0), cache.ttl)
}

/******************************************
 * getTTL
 ******************************************/

// TestGetTTL covers the three-step fallback: the response's own max-age, then the
// cache's default, then one minute.
func TestGetTTL(t *testing.T) {

	withDefault, _ := newTestCache(WithTTL(5 * time.Minute))
	noDefault, _ := newTestCache()

	// check asserts the TTL that one cache derives from one Cache-Control value
	check := func(cache HTTPCache, cacheControl string, expected time.Duration) {
		t.Helper()
		require.Equal(t, expected, cache.getTTL(newTestResponse("", "Cache-Control", cacheControl)))
	}

	// The response's own max-age wins over everything
	check(withDefault, "max-age=30", 30*time.Second)
	check(noDefault, "max-age=30", 30*time.Second)
	check(withDefault, "public, max-age=120, must-revalidate", 120*time.Second)

	// With no usable max-age, the cache's own default applies
	check(withDefault, "", 5*time.Minute)
	check(withDefault, "no-cache", 5*time.Minute)
	check(withDefault, "max-age=0", 5*time.Minute)

	// With neither, one minute
	check(noDefault, "", time.Minute)
	check(noDefault, "max-age=0", time.Minute)
	check(noDefault, "garbage", time.Minute)
}

// TestGetTTL_NegativeMaxAge confirms a negative max-age is ignored rather than producing
// a negative TTL, which an adapter would treat as already expired or as never expiring.
func TestGetTTL_NegativeMaxAge(t *testing.T) {

	cache, _ := newTestCache(WithTTL(time.Minute))
	require.Equal(t, time.Minute, cache.getTTL(newTestResponse("", "Cache-Control", "max-age=-5")))
}

/******************************************
 * Metadata
 ******************************************/

// TestGetResponseMetadata covers which response headers are recorded as cache metadata.
func TestGetResponseMetadata(t *testing.T) {

	cache, _ := newTestCache()

	t.Run("Empty", func(t *testing.T) {
		require.Equal(t, url.Values{}, cache.getResponseMetadata(newTestResponse("")))
	})

	t.Run("ETagOnly", func(t *testing.T) {
		metadata := cache.getResponseMetadata(newTestResponse("", "ETag", `"abc123"`))
		require.Equal(t, `"abc123"`, metadata.Get("ETag"))
		require.Equal(t, "", metadata.Get("Vary"))
	})

	t.Run("VaryRecordsTheNamedHeaderValues", func(t *testing.T) {
		response := newTestResponse("", "Vary", "Accept, Accept-Language", "Accept", "text/html", "Accept-Language", "en-US")
		metadata := cache.getResponseMetadata(response)

		require.Equal(t, "Accept, Accept-Language", metadata.Get("Vary"))
		require.Equal(t, "text/html", metadata.Get("Accept"))
		require.Equal(t, "en-US", metadata.Get("Accept-Language"))
	})

	t.Run("VaryNamesAHeaderTheResponseLacks", func(t *testing.T) {
		metadata := cache.getResponseMetadata(newTestResponse("", "Vary", "Accept-Encoding"))
		require.Equal(t, "", metadata.Get("Accept-Encoding"), "an absent header records as empty, not missing")
	})
}

// TestGetMetadata covers reading metadata back out of storage, including the two ways
// it can be absent.
func TestGetMetadata(t *testing.T) {

	cache, adapter := newTestCache()

	t.Run("Miss", func(t *testing.T) {
		metadata, ok := cache.getMetadata("https://example.com/missing")
		require.False(t, ok)
		require.Nil(t, metadata)
	})

	t.Run("Hit", func(t *testing.T) {
		adapter["https://example.com/page"+metadataMarker] = "ETag=%22abc%22&Vary=Accept"
		metadata, ok := cache.getMetadata("https://example.com/page")

		require.True(t, ok)
		require.Equal(t, `"abc"`, metadata.Get("ETag"))
		require.Equal(t, "Accept", metadata.Get("Vary"))
	})

	t.Run("MalformedRecord", func(t *testing.T) {
		adapter["https://example.com/bad"+metadataMarker] = "%zz"
		metadata, ok := cache.getMetadata("https://example.com/bad")

		require.False(t, ok, "an unparseable record is a miss, not a panic")
		require.Nil(t, metadata)
	})

	t.Run("EmptyRecord", func(t *testing.T) {
		adapter["https://example.com/empty"+metadataMarker] = ""
		metadata, ok := cache.getMetadata("https://example.com/empty")

		require.True(t, ok, "an empty query string parses to empty Values")
		require.Equal(t, url.Values{}, metadata)
	})
}

/******************************************
 * getVariesValues
 ******************************************/

// TestGetVariesValues covers the request-header fingerprint that separates one cached
// variant from another.
func TestGetVariesValues(t *testing.T) {

	cache, _ := newTestCache()

	t.Run("NoVaryStillFingerprintsTheEmptyName", func(t *testing.T) {
		request := newTestRequest(t, "https://example.com/")
		// Splitting "" yields one empty fieldname, so the result is never truly empty
		require.Equal(t, "=", cache.getVariesValues(request, url.Values{}))
	})

	t.Run("SingleField", func(t *testing.T) {
		request := newTestRequest(t, "https://example.com/", "Accept", "text/html")
		metadata := url.Values{"Vary": []string{"Accept"}}

		require.Equal(t, "Accept=text%2Fhtml", cache.getVariesValues(request, metadata))
	})

	t.Run("MultipleFieldsAreOrderIndependent", func(t *testing.T) {
		metadata := url.Values{"Vary": []string{"Accept, Accept-Language"}}

		first := cache.getVariesValues(newTestRequest(t, "https://example.com/", "Accept", "text/html", "Accept-Language", "en"), metadata)
		second := cache.getVariesValues(newTestRequest(t, "https://example.com/", "Accept-Language", "en", "Accept", "text/html"), metadata)

		require.Equal(t, first, second, "url.Values.Encode sorts by key")
	})

	t.Run("DifferentValuesFingerprintDifferently", func(t *testing.T) {
		metadata := url.Values{"Vary": []string{"Accept"}}

		html := cache.getVariesValues(newTestRequest(t, "https://example.com/", "Accept", "text/html"), metadata)
		json := cache.getVariesValues(newTestRequest(t, "https://example.com/", "Accept", "application/json"), metadata)

		require.NotEqual(t, html, json)
	})

	t.Run("MissingRequestHeaderIsEmptyNotAbsent", func(t *testing.T) {
		request := newTestRequest(t, "https://example.com/")
		metadata := url.Values{"Vary": []string{"Accept"}}

		require.Equal(t, "Accept=", cache.getVariesValues(request, metadata))
	})
}

/******************************************
 * setResponse and getResponse
 ******************************************/

// TestResponseRoundTrip stores a response and reads it back, which is the whole point
// of the package.
func TestResponseRoundTrip(t *testing.T) {

	cache, _ := newTestCache()
	request := newTestRequest(t, "https://example.com/page")

	cache.setResponse(request, newTestResponse("Hello, world", "Content-Type", "text/plain"))

	response, ok := cache.getResponse(request)
	require.True(t, ok)
	require.NotNil(t, response)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "text/plain", response.Header.Get("Content-Type"))

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello, world", string(body))
}

// TestSetResponse_MarksTheCachedCopy confirms the X-Cache header is added, which is how
// a cached response is told apart from a live one when troubleshooting.
func TestSetResponse_MarksTheCachedCopy(t *testing.T) {

	cache, _ := newTestCache()
	request := newTestRequest(t, "https://example.com/page")

	cache.setResponse(request, newTestResponse("body"))

	response, ok := cache.getResponse(request)
	require.True(t, ok)
	require.Equal(t, "HIT from HTTPCache", response.Header.Get("X-Cache"))
}

// TestSetResponse_WritesTwoKeys confirms the metadata and the body are stored separately,
// because the metadata must be readable before the body's key can be computed.
func TestSetResponse_WritesTwoKeys(t *testing.T) {

	cache, adapter := newTestCache()
	request := newTestRequest(t, "https://example.com/page")

	cache.setResponse(request, newTestResponse("body"))

	require.Equal(t, 2, len(adapter))
	require.Contains(t, adapter, "https://example.com/page"+metadataMarker)
}

// TestGetResponse_MissingMetadata confirms a body with no metadata record is a miss,
// since the body's key cannot be derived without it.
func TestGetResponse_MissingMetadata(t *testing.T) {

	cache, _ := newTestCache()

	response, ok := cache.getResponse(newTestRequest(t, "https://example.com/nothing"))
	require.False(t, ok)
	require.Nil(t, response)
}

// TestGetResponse_MissingBody confirms metadata without a matching body is a miss.
func TestGetResponse_MissingBody(t *testing.T) {

	cache, adapter := newTestCache()
	adapter["https://example.com/page"+metadataMarker] = ""

	response, ok := cache.getResponse(newTestRequest(t, "https://example.com/page"))
	require.False(t, ok)
	require.Nil(t, response)
}

// TestGetResponse_UnparseableBody confirms a stored record that is not a valid HTTP
// response is a miss rather than an error or a panic.
func TestGetResponse_UnparseableBody(t *testing.T) {

	cache, adapter := newTestCache()
	request := newTestRequest(t, "https://example.com/page")

	adapter["https://example.com/page"+metadataMarker] = ""
	adapter["https://example.com/page"+headSeparator+cache.getVariesValues(request, url.Values{})] = "this is not an HTTP response"

	response, ok := cache.getResponse(request)
	require.False(t, ok)
	require.Nil(t, response)
}

// TestResponseRoundTrip_VaryIsolatesVariants is the behavior Vary exists for: two requests
// that differ only in a Vary header must not read each other's cached body.
func TestResponseRoundTrip_VaryIsolatesVariants(t *testing.T) {

	cache, _ := newTestCache()

	htmlRequest := newTestRequest(t, "https://example.com/page", "Accept", "text/html")
	jsonRequest := newTestRequest(t, "https://example.com/page", "Accept", "application/json")

	cache.setResponse(htmlRequest, newTestResponse("<html>", "Vary", "Accept", "Content-Type", "text/html"))

	// The HTML variant is readable...
	response, ok := cache.getResponse(htmlRequest)
	require.True(t, ok)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "<html>", string(body))

	// ...and the JSON variant is a MISS, not the HTML body
	jsonResponse, ok := cache.getResponse(jsonRequest)
	require.False(t, ok, "a different Accept header must not read the HTML variant")
	require.Nil(t, jsonResponse)
}

// TestResponseRoundTrip_SameURLDifferentQuery confirms the query string is part of the key.
func TestResponseRoundTrip_SameURLDifferentQuery(t *testing.T) {

	cache, _ := newTestCache()

	cache.setResponse(newTestRequest(t, "https://example.com/page?a=1"), newTestResponse("first"))

	response, ok := cache.getResponse(newTestRequest(t, "https://example.com/page?a=2"))
	require.False(t, ok)
	require.Nil(t, response)
}

/******************************************
 * Failing bodies
 *
 * An upstream response whose body errors part-way through is the
 * realistic case these guard: a truncated or reset connection.
 ******************************************/

// failingBody is an io.ReadCloser that errors instead of returning data, standing in for a
// response body that dies mid-read.
type failingBody struct{}

// Read implements io.Reader, always failing.
func (failingBody) Read(_ []byte) (int, error) {
	return 0, errors.New("connection reset")
}

// Close implements io.Closer.
func (failingBody) Close() error {
	return nil
}

// TestSetResponse_UnreadableBody confirms that a body which cannot be read aborts the store
// without panicking, and leaves only the metadata key behind.
func TestSetResponse_UnreadableBody(t *testing.T) {

	cache, adapter := newTestCache()
	request := newTestRequest(t, "https://example.com/page")

	response := newTestResponse("")
	response.Body = failingBody{}
	response.ContentLength = 64

	cache.setResponse(request, response)

	// The metadata is written before the body is read, so it survives; the body does not
	require.Contains(t, adapter, "https://example.com/page"+metadataMarker)
	require.Equal(t, 1, len(adapter), "a body that cannot be read must not be stored")

	cached, ok := cache.getResponse(request)
	require.False(t, ok, "the half-written entry must read as a MISS")
	require.Nil(t, cached)
}
