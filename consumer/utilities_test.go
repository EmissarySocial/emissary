package consumer

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestGetHostnameFromArgs documents the contract that every WithFactory-backed
// task enqueue must satisfy: WithFactory resolves the tenant Factory from the
// hostname returned here, and hard-fails the task when it is empty. A task
// enqueued with only a "url" argument (as the reply-tree crawlers once were)
// yields no hostname and can never run.
func TestGetHostnameFromArgs(t *testing.T) {

	// A "host" argument is used directly (reduced to its hostname).
	require.Equal(t, "example.com", getHostnameFromArgs(mapof.Any{"host": "https://example.com/@alice"}))
	require.Equal(t, "example.com", getHostnameFromArgs(mapof.Any{"host": "example.com"}))

	// An "actor" argument is used as a fallback source of the hostname.
	require.Equal(t, "example.com", getHostnameFromArgs(mapof.Any{"actor": "https://example.com/@alice"}))

	// "host" takes precedence over "actor".
	require.Equal(t, "host.example", getHostnameFromArgs(mapof.Any{
		"host":  "https://host.example/@alice",
		"actor": "https://actor.example/@bob",
	}))

	// A "url"-only map (the reply-tree crawler bug) yields NO hostname, which is
	// what caused those tasks to hard-fail with "Missing 'host' argument".
	require.Equal(t, "", getHostnameFromArgs(mapof.Any{"url": "http://localhost/6a4daccdc20bb1b4d44b8f94"}))

	// An empty map yields no hostname.
	require.Equal(t, "", getHostnameFromArgs(mapof.Any{}))
}
