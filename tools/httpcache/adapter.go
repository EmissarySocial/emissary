package httpcache

import "time"

type Adapter interface {
	Get(key string) (string, bool)
	Set(key string, value string, ttl time.Duration) bool
	Delete(key string)
	Close()
}
