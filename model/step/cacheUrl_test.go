package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestCacheURL verifies that a "cache-url" step parses its configuration
func TestCacheURL(t *testing.T) {

	// Public cache with an explicit max-age.
	step, err := NewCacheURL(mapof.Any{"max-age": 60})
	require.Nil(t, err)
	require.Equal(t, "public, max-age=60", step.CacheControl)

	// Private cache.
	step, err = NewCacheURL(mapof.Any{"private": true, "max-age": 120})
	require.Nil(t, err)
	require.Equal(t, "private, max-age=120", step.CacheControl)

	// max-age defaults to 3600.
	step, err = NewCacheURL(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "public, max-age=3600", step.CacheControl)

	require.Equal(t, "cache-url", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}
