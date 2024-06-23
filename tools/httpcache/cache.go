package httpcache

import (
	"bufio"
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/rs/zerolog/log"
)

type HTTPCache struct {
	Adapter
	ttl time.Duration
}

/*
cache rules:

only cache GET requests
cache TTL
support Vary modifiers
support ETags

*/

func (cache *HTTPCache) With(options ...Option) {
	for _, option := range options {
		option(cache)
	}
}

// setResponse stores a response in the cache, along with the metadata required to
// support Vary and ETag headers.
// IMPORTANT: This function will close the response body, so it must only be called
// with a COPY of the original response, or else the calling application will not be able
// to read the response body.
func (cache *HTTPCache) setResponse(request *http.Request, response *http.Response) {

	// Get the metadata for the request
	address := request.URL.String()
	ttl := cache.getTTL(response)
	cacheKey := request.URL.String() + metadataMarker
	metadata := cache.getResponseMetadata(response)

	log.Trace().Str("url", cacheKey).Str("metadata", metadata.Encode()).Msg("Setting Response")

	cache.Set(cacheKey, metadata.Encode(), ttl)

	// Write the response into a buffer
	var buffer bytes.Buffer
	response.Header.Set("X-Cache", "HIT from HTTPCache") // Mark the cached value for troubleshooting
	if err := response.Write(&buffer); err != nil {
		return
	}

	// Save the actual response into the cache
	cacheKey = address + headSeparator + cache.getVariesValues(request, metadata)
	cache.Set(cacheKey, buffer.String(), ttl)
}

// getResponse retrieves a cached response for the given request.  If no response
// is found, the second return value will be false.
func (cache *HTTPCache) getResponse(request *http.Request) (*http.Response, bool) {

	// Get the metadata for the request
	metadata, ok := cache.getMetadata(request.URL.String())

	if !ok {
		return nil, false
	}

	// Find the header fields that make this request unique
	variesValues := cache.getVariesValues(request, metadata)
	cacheKey := request.URL.String() + headSeparator + variesValues

	record, ok := cache.Get(cacheKey)

	if !ok {
		return nil, false
	}

	// Read the record into an HTTP response
	reader := bufio.NewReader(strings.NewReader(record))
	response, err := http.ReadResponse(reader, request)

	if err != nil {
		return nil, false
	}

	// Success!
	return response, true
}

// getVariesValues retrieves all
func (cache *HTTPCache) getVariesValues(request *http.Request, metadata url.Values) string {

	result := url.Values{}

	fields := strings.Split(metadata.Get("Vary"), ",")
	for _, fieldname := range fields {
		fieldname = strings.TrimSpace(fieldname)
		value := request.Header.Get(fieldname)
		result.Set(fieldname, value)
	}

	return result.Encode()
}

func (cache *HTTPCache) getTTL(response *http.Response) time.Duration {

	header := cacheheader.ParseString(response.Header.Get("Cache-Control"))

	if header.MaxAge > 0 {
		return time.Duration(header.MaxAge) * time.Second
	}

	if cache.ttl > 0 {
		return cache.ttl
	}

	return 1 * time.Minute
}

/******************************************
 * Metadata manipulations
 ******************************************/

// getResponseMetadata returns all of the caching metadata required for a response.
func (cache *HTTPCache) getResponseMetadata(response *http.Response) url.Values {

	result := url.Values{}

	if etag := response.Header.Get("ETag"); etag != "" {
		result.Set("ETag", etag)
	}

	if varies := response.Header.Get("Vary"); varies != "" {
		result.Set("Vary", varies)

		for _, fieldname := range strings.Split(varies, ",") {
			fieldname = strings.TrimSpace(fieldname)
			value := response.Header.Get(fieldname)
			result.Set(fieldname, value)
		}
	}

	return result

}

// getMetadata retrieves the header record for a URL.  This *should* contain
// the ETag value and Varies header values.
func (cache *HTTPCache) getMetadata(address string) (url.Values, bool) {

	cacheKey := address + metadataMarker

	if record, ok := cache.Get(cacheKey); ok {
		if result, err := url.ParseQuery(record); err == nil {
			return result, true
		}
	}

	return nil, false
}
