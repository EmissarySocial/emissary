package httpcache

import "time"

// Option is a configuration function that modifies an HTTPCache
type Option func(*HTTPCache)

// WithTTL sets the default TTL for this cache, which is used
// if the response does not contain a Cache-Control header.
func WithTTL(ttl time.Duration) Option {
	return func(cache *HTTPCache) {
		cache.ttl = ttl
	}
}
