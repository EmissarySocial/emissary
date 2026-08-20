package httpcache

import "time"

// Adapter is the storage engine that an HTTPCache reads and writes through
type Adapter interface {
	Get(key string) (string, bool)
	Set(key string, value string, ttl time.Duration) bool
	Delete(key string)
	Close()
}
