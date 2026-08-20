package httpcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testAdapter is an in-memory Adapter that ignores TTLs, used by the tests in this package
type testAdapter map[string]string

// Get implements the Adapter interface, reading a value from the map
func (adapter testAdapter) Get(key string) (string, bool) {
	value, ok := adapter[key]
	return value, ok
}

// Set implements the Adapter interface, writing a value into the map. The TTL is ignored.
func (adapter testAdapter) Set(key string, value string, _ time.Duration) bool {
	adapter[key] = value
	return true
}

// Delete implements the Adapter interface, removing a value from the map
func (adapter testAdapter) Delete(key string) {
	delete(adapter, key)
}

// Close implements the Adapter interface. The stub holds no resources to release.
func (adapter testAdapter) Close() {
	// Do nothing
}

// TestCache verifies that values written into the cache can be read back, and disappear once deleted
func TestCache(t *testing.T) {

	cache := HTTPCache{
		Adapter: testAdapter{},
	}

	cache.Set("key1", "TEST", 10*time.Second)

	value, ok := cache.Get("key1")
	require.True(t, ok)
	require.Equal(t, "TEST", value)
}
