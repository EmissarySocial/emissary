package httpcache

import "time"

// OtterCache mimics the public API of the Otter cache so that we don't have to
// import it into this library.
// https://github.com/maypok86/otter
type OtterCache interface {
	Get(key string) (string, bool)
	Set(key string, value string, ttl time.Duration) bool
	Delete(key string)
	Close()
}

func NewOtterCache(cache OtterCache, options ...Option) HTTPCache {
	result := HTTPCache{
		Adapter: cache,
	}
	result.With(options...)
	return result
}
