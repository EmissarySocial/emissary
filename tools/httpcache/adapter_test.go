package httpcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testAdapter map[string]string

func (adapter testAdapter) Get(key string) (string, bool) {
	value, ok := adapter[key]
	return value, ok
}

func (adapter testAdapter) Set(key string, value string, _ time.Duration) bool {
	adapter[key] = value
	return true
}

func (adapter testAdapter) Delete(key string) {
	delete(adapter, key)
}

func (adapter testAdapter) Close() {
	// Do nothing
}

func TestCache(t *testing.T) {

	cache := HTTPCache{
		Adapter: testAdapter{},
	}

	cache.Set("key1", "TEST", 10*time.Second)

	value, ok := cache.Get("key1")
	require.True(t, ok)
	require.Equal(t, "TEST", value)
}
