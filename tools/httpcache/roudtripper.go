package httpcache

import (
	"net/http"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/re"
	"github.com/rs/zerolog/log"
)

// RoundTripper implements the http.RoundTripper interface, that is used by the http.Client to cache outbound HTTP requests.
// https://echorand.me/posts/go-http-client-middleware/
// https://lanre.wtf/blog/2017/07/24/roundtripper-go
// https://pkg.go.dev/net/http#RoundTripper
type RoundTripper struct {
	cache  *HTTPCache   // cache to use before/after HTTP requests
	client *http.Client // inner HTTP client to use for HTTP requests
}

// NewHTTPClient returns a fully initialized http.Client that uses the provided cache
// to store responses.  It uses the default http.Client as a base, and replaces the
// transport with a new, caching RoundTripper.
func NewHTTPClient(cache *HTTPCache) *http.Client {
	return &http.Client{
		Transport: NewRoundTripper(cache),
	}
}

// NewRoundTripper returns a fully initialized RoundTripper
func NewRoundTripper(cache *HTTPCache) RoundTripper {

	return RoundTripper{
		cache: cache,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetCache replaces the existing cache with a new one
func (roundTripper *RoundTripper) SetCache(cache *HTTPCache) {
	roundTripper.cache = cache
}

// SetClient replaces the default http.Client with a custom client
func (roundTripper *RoundTripper) SetClient(client *http.Client) {
	roundTripper.client = client
}

// RoundTrip implements the http.RoundTripper interface, which replaces the http.Client's default
// behavior with a caching mechanism.
func (roundTripper RoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {

	// If this is not a GET request, then skip the cache altogether
	if request.Method != http.MethodGet {
		return roundTripper.client.Do(request)
	}

	// Check the cache for a response
	response, found := roundTripper.cache.getResponse(request) // nolint:scopeguard

	if found {
		log.Trace().Str("url", request.URL.String()).Msg("HTTPCache: Cache HIT")
		return response, nil
	}

	log.Trace().Str("url", request.URL.String()).Msg("HTTPCache: Cache MISS")

	// Fall through means that we actually need to do the HTTP request
	response, err := roundTripper.client.Do(request)

	if err != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}
		return response, derp.Wrap(err, "httpcache.RoundTripper.RoundTrip", "Error executing HTTP request", derp.WithCode(statusCode))
	}

	// Save the response to the cache
	responseCopy := re.CloneResponse(response)
	roundTripper.cache.setResponse(request, &responseCopy)

	// Return response to the caller
	return response, nil
}
