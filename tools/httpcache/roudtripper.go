package httpcache

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/re"
	"github.com/rs/zerolog/log"
)

// RoundTripper implements the http.RoundTripper interface, that is used by the http.Client to cache outbound HTTP requests.
// https://echorand.me/posts/go-http-client-middleware/
// https://lanre.wtf/blog/2017/07/24/roundtripper-go
// https://pkg.go.dev/net/http#RoundTripper
type RoundTripper struct {
	client *http.Client
	cache  *HTTPCache
}

// NewHTTPClient returns a fully initialized http.Client that uses the provided cache
// to store responses.  It uses the default http.Client as a base, and replaces the
// transport with a new, caching RoundTripper.
func NewHTTPClient(client *http.Client, cache *HTTPCache) *http.Client {

	roundTripper := NewRoundTripper(client, cache)

	return &http.Client{
		Timeout:       client.Timeout,
		CheckRedirect: client.CheckRedirect,
		Jar:           client.Jar,
		Transport:     roundTripper,
	}
}

// NewRoundTripper returns a fully initialized RoundTripper
func NewRoundTripper(client *http.Client, cache *HTTPCache) RoundTripper {
	return RoundTripper{
		client: client,
		cache:  cache,
	}
}

func (roundTripper RoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {

	// Check the cache for a response
	response, ok := roundTripper.cache.getResponse(request)

	if ok {
		log.Trace().Str("url", request.URL.String()).Msg("HTTPCache: Cache HIT")
		return response, nil
	}

	log.Trace().Str("url", request.URL.String()).Msg("HTTPCache: Cache MISS")

	// Fall through means that we actually need to do the HTTP request
	response, err := roundTripper.client.Do(request)

	if err != nil {
		return response, derp.Wrap(err, "httpcache.RoundTripper.RoundTrip", "Error executing HTTP request", derp.WithCode(response.StatusCode))
	}

	// Save the response to the cache
	responseCopy := re.CloneResponse(response)
	roundTripper.cache.setResponse(request, &responseCopy)

	// Return response to the caller
	return response, nil
}
