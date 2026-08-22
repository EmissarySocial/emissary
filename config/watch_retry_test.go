package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWatchRetryDelay pins the backoff schedule for reopening a dead watcher: it doubles, and it
// is capped.  The cap is the load-bearing part -- an uncapped backoff eventually schedules the
// next attempt so far out that a node is functionally dead anyway.
func TestWatchRetryDelay(t *testing.T) {

	table := []struct {
		failures int
		expected time.Duration
	}{
		{0, 1 * time.Second}, // defensive: never returns zero, which would hot-loop
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 16 * time.Second},
		{6, 32 * time.Second},
		{7, watchRetryMaximum},
		{8, watchRetryMaximum},
		{1000, watchRetryMaximum},
	}

	for _, test := range table {
		require.Equal(t, test.expected, watchRetryDelay(test.failures), "failures=%d", test.failures)
	}
}
