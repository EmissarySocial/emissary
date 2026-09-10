package httpcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestNewOtterCache confirms the constructor stores the supplied cache as the Adapter and
// applies its options. testAdapter satisfies OtterCache, which is the same method set.
func TestNewOtterCache(t *testing.T) {

	adapter := testAdapter{}
	cache := NewOtterCache(adapter, WithTTL(3*time.Minute))

	require.Equal(t, 3*time.Minute, cache.ttl)

	cache.Set("key", "value", time.Second)
	require.Equal(t, "value", adapter["key"], "the Otter cache IS the Adapter")
}

// TestNewOtterCache_NoOptions confirms the constructor works with no options, leaving the
// zero TTL for getTTL to fall back from.
func TestNewOtterCache_NoOptions(t *testing.T) {

	cache := NewOtterCache(testAdapter{})
	require.Equal(t, time.Duration(0), cache.ttl)
}

// TestOtterCacheInterface pins that the Adapter and OtterCache method sets are identical,
// which is what lets an Otter cache be passed without importing the library.
func TestOtterCacheInterface(t *testing.T) {

	var adapter Adapter = testAdapter{}
	otter, ok := adapter.(OtterCache)

	require.True(t, ok, "every Adapter must satisfy OtterCache")
	require.NotNil(t, otter)
}
