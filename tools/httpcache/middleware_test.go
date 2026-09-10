package httpcache

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeRoundTripper stands in for the next RoundTripper in the chain, counting calls so a
// test can prove whether the cache was consulted.
type fakeRoundTripper struct {
	calls    int
	response *http.Response
	err      error
}

// RoundTrip implements http.RoundTripper, returning the canned response and counting the call.
func (fake *fakeRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {

	fake.calls++

	if fake.err != nil {
		return fake.response, fake.err
	}

	return fake.response, nil
}

// newTestMiddleware wires a caching middleware around a fake transport, and returns all three
// so a test can inspect the cache and the call count.
func newTestMiddleware(t *testing.T, response *http.Response, err error) (http.RoundTripper, *fakeRoundTripper, *HTTPCache) {

	t.Helper()

	cache, _ := newTestCache()
	fake := &fakeRoundTripper{response: response, err: err}

	return NewHTTPMiddleware(&cache)(fake), fake, &cache
}

// TestNewHTTPMiddleware confirms the constructor returns a RoundTripper wrapper.
func TestNewHTTPMiddleware(t *testing.T) {

	cache, _ := newTestCache()
	wrapped := NewHTTPMiddleware(&cache)(&fakeRoundTripper{})

	require.NotNil(t, wrapped)
	require.IsType(t, HTTPMiddleware{}, wrapped)
}

// TestRoundTrip_CacheMissThenHit is the core behavior: the first GET reaches the transport,
// and the second is served from the cache without touching it.
func TestRoundTrip_CacheMissThenHit(t *testing.T) {

	middleware, fake, _ := newTestMiddleware(t, newTestResponse("payload", "Content-Type", "text/plain"), nil)
	request := newTestRequest(t, "https://example.com/page")

	// MISS: the transport is called, and the caller can still read the body
	first, err := middleware.RoundTrip(request)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.Equal(t, 1, fake.calls)

	body, err := io.ReadAll(first.Body)
	require.NoError(t, err)
	require.Equal(t, "payload", string(body), "caching must not consume the caller's body")

	// HIT: the transport is NOT called again
	second, err := middleware.RoundTrip(request)
	require.NoError(t, err)
	require.NotNil(t, second)
	require.Equal(t, 1, fake.calls, "a cached response must not reach the transport")
	require.Equal(t, "HIT from HTTPCache", second.Header.Get("X-Cache"))

	cached, err := io.ReadAll(second.Body)
	require.NoError(t, err)
	require.Equal(t, "payload", string(cached))
}

// TestRoundTrip_NonGetBypassesTheCache covers the method guard. Only GET is cacheable, so
// every other method must reach the transport every time and store nothing.
func TestRoundTrip_NonGetBypassesTheCache(t *testing.T) {

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead} {

		t.Run(method, func(t *testing.T) {

			middleware, fake, cache := newTestMiddleware(t, newTestResponse("payload"), nil)

			request, err := http.NewRequest(method, "https://example.com/page", nil)
			require.NoError(t, err)

			_, err = middleware.RoundTrip(request)
			require.NoError(t, err)

			_, err = middleware.RoundTrip(request)
			require.NoError(t, err)

			require.Equal(t, 2, fake.calls, "%s must never be served from the cache", method)

			adapter, ok := cache.Adapter.(testAdapter)
			require.True(t, ok)
			require.Equal(t, 0, len(adapter), "%s must not be stored", method)
		})
	}
}

// TestRoundTrip_TransportError confirms a failed request is wrapped and returned, and that
// nothing is written to the cache.
func TestRoundTrip_TransportError(t *testing.T) {

	middleware, fake, cache := newTestMiddleware(t, nil, errors.New("connection refused"))

	response, err := middleware.RoundTrip(newTestRequest(t, "https://example.com/page"))

	require.NotNil(t, err)
	require.Nil(t, response)
	require.Equal(t, 1, fake.calls)

	adapter, ok := cache.Adapter.(testAdapter)
	require.True(t, ok)
	require.Equal(t, 0, len(adapter), "a failed request must not be cached")
}

// TestRoundTrip_TransportErrorWithResponse confirms a transport that returns both a response
// and an error still surfaces the error, and still caches nothing.
func TestRoundTrip_TransportErrorWithResponse(t *testing.T) {

	failed := newTestResponse("gateway timeout")
	failed.StatusCode = http.StatusGatewayTimeout

	middleware, _, cache := newTestMiddleware(t, failed, errors.New("upstream failed"))

	response, err := middleware.RoundTrip(newTestRequest(t, "https://example.com/page"))

	require.NotNil(t, err)
	require.NotNil(t, response, "the response is passed through alongside the error")

	adapter, ok := cache.Adapter.(testAdapter)
	require.True(t, ok)
	require.Equal(t, 0, len(adapter))
}

// TestRoundTrip_DifferentURLsAreCachedSeparately confirms the cache key includes the URL,
// so one page's body is never served for another.
func TestRoundTrip_DifferentURLsAreCachedSeparately(t *testing.T) {

	middleware, fake, _ := newTestMiddleware(t, newTestResponse("payload"), nil)

	_, err := middleware.RoundTrip(newTestRequest(t, "https://example.com/first"))
	require.NoError(t, err)

	// A second URL is a MISS even though the first is cached
	fake.response = newTestResponse("payload")
	_, err = middleware.RoundTrip(newTestRequest(t, "https://example.com/second"))
	require.NoError(t, err)

	require.Equal(t, 2, fake.calls)
}

// TestRoundTrip_VaryIsHonored confirms the middleware respects the Vary header end to end:
// a request differing only in a Vary header is a MISS.
func TestRoundTrip_VaryIsHonored(t *testing.T) {

	middleware, fake, _ := newTestMiddleware(t, newTestResponse("<html>", "Vary", "Accept"), nil)

	_, err := middleware.RoundTrip(newTestRequest(t, "https://example.com/page", "Accept", "text/html"))
	require.NoError(t, err)
	require.Equal(t, 1, fake.calls)

	// Same URL, different Accept: must reach the transport again
	fake.response = newTestResponse("{}", "Vary", "Accept")
	_, err = middleware.RoundTrip(newTestRequest(t, "https://example.com/page", "Accept", "application/json"))
	require.NoError(t, err)
	require.Equal(t, 2, fake.calls)

	// ...and the original variant is still a HIT
	_, err = middleware.RoundTrip(newTestRequest(t, "https://example.com/page", "Accept", "text/html"))
	require.NoError(t, err)
	require.Equal(t, 2, fake.calls, "the HTML variant must still be cached")
}

// TestRoundTrip_UncloneableResponse confirms that a response whose body cannot be read is
// returned to the caller with a wrapped error, rather than being cached or swallowed.
func TestRoundTrip_UncloneableResponse(t *testing.T) {

	broken := newTestResponse("")
	broken.Body = failingBody{}
	broken.ContentLength = 64

	middleware, fake, cache := newTestMiddleware(t, broken, nil)

	response, err := middleware.RoundTrip(newTestRequest(t, "https://example.com/page"))

	require.NotNil(t, err, "a body that cannot be cloned must be reported")
	require.NotNil(t, response, "the caller still receives the original response")
	require.Equal(t, 1, fake.calls)

	adapter, ok := cache.Adapter.(testAdapter)
	require.True(t, ok)
	require.Equal(t, 0, len(adapter), "nothing is cached when the clone fails")
}
