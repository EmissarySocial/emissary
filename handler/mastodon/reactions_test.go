package mastodon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReactorLimit confirms a client's page size is clamped to Mastodon's documented range
func TestReactorLimit(t *testing.T) {

	require.Equal(t, reactorDefaultLimit, reactorLimit(0))
	require.Equal(t, reactorDefaultLimit, reactorLimit(-1))
	require.Equal(t, 10, reactorLimit(10))
	require.Equal(t, reactorMaxLimit, reactorLimit(80))
	require.Equal(t, reactorMaxLimit, reactorLimit(500))
}
