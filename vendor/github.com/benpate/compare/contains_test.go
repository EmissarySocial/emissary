package compare

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContains(t *testing.T) {
	require.True(t, Contains([]string{"hello"}, "hello"))
	require.True(t, Contains([]string{"hello", "there", "general", "kenobi"}, "hello"))
	require.True(t, Contains([]string{"hello", "there", "general", "kenobi"}, "there"))
	require.True(t, Contains([]string{"hello", "there", "general", "kenobi"}, "general"))
	require.True(t, Contains([]string{"hello", "there", "general", "kenobi"}, "kenobi"))
	require.False(t, Contains([]string{"hello", "there", "general", "kenobi"}, "grievous"))
}
