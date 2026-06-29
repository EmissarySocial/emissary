package httpcache

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/re"
	"github.com/rs/zerolog/log"
)

// HTTPMiddleware implements the http.RoundTripper interface, that is used by the http.Client to cache outbound HTTP requests.
// https://echorand.me/posts/go-http-client-middleware/
// https://lanre.wtf/blog/2017/07/24/roundtripper-go
// https://pkg.go.dev/net/http#RoundTripper
type HTTPMiddleware struct {
	cache *HTTPCache        // cache to use before/after HTTP requests
	next  http.RoundTripper // next RoundTripper in the chain
}

// NewHTTPMiddleware returns a function that wraps an existing http.RoundTripper with caching functionality.
func NewHTTPMiddleware(cache *HTTPCache) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return HTTPMiddleware{
			cache: cache,
			next:  next,
		}
	}
}

// RoundTrip implements the http.RoundTripper interface, which replaces the http.Client's default
// behavior with a caching mechanism.
func (middleware HTTPMiddleware) RoundTrip(request *http.Request) (*http.Response, error) {

	const location = "httpcache.HTTPMiddleware.RoundTrip"

	// If this is not a GET request, then skip the cache altogether
	if request.Method != http.MethodGet {
		return middleware.next.RoundTrip(request)
	}

	// Check the cache for a response
	response, found := middleware.cache.getResponse(request) // nolint:scopeguard

	if found {
		log.Trace().Str("url", request.URL.String()).Msg("HTTPCache: Cache HIT")
		return response, nil
	}

	log.Trace().Str("url", request.URL.String()).Msg("HTTPCache: Cache MISS")

	// Fall through means that we actually need to do the HTTP request
	response, err := middleware.next.RoundTrip(request)

	if err != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}
		return response, derp.Wrap(err, location, "Error executing HTTP request", derp.WithCode(statusCode))
	}

	// Save the response to the cache (body capped to guard against an oversized response)
	responseCopy, err := re.CloneResponse(response, re.DefaultMaximum)

	if err != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}
		return response, derp.Wrap(err, location, "Error cloning HTTP response", derp.WithCode(statusCode))
	}

	middleware.cache.setResponse(request, &responseCopy)

	// Return response to the caller
	return response, nil
}
