package mastodon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCheckProfileImage confirms only reasonably sized images are accepted as profile pictures
func TestCheckProfileImage(t *testing.T) {

	require.NoError(t, checkProfileImage("image/png", 1024))
	require.NoError(t, checkProfileImage("image/jpeg", profileImageMaxBytes))
	require.NoError(t, checkProfileImage("image/webp", 0))

	require.Error(t, checkProfileImage("text/html; charset=utf-8", 1024), "a page is not an image")
	require.Error(t, checkProfileImage("application/octet-stream", 1024), "unknown bytes are not an image")
	require.Error(t, checkProfileImage("image/png", profileImageMaxBytes+1), "too large")
}
