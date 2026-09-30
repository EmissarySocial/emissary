package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStream_Toot_Sensitive confirms a post with a content warning is reported as sensitive
func TestStream_Toot_Sensitive(t *testing.T) {

	withWarning := NewStream()
	withWarning.Label = "Spoilers ahead"

	status := withWarning.Toot()
	require.Equal(t, "Spoilers ahead", status.SpoilerText)
	require.True(t, status.Sensitive)

	plain := NewStream().Toot()
	require.Empty(t, plain.SpoilerText)
	require.False(t, plain.Sensitive)
}
