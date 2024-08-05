package camper

import "time"

// Cache defines an interface for caching key/value pairs
// This (mysteriously) exactly matches the httpcache.Adapter interface
type Cache interface {
	Get(key string) (string, bool)
	Set(key string, value string, ttl time.Duration) bool
	Delete(key string)
	Close()
}
