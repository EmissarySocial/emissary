package tabular

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJoinPath(t *testing.T) {

	require.Equal(t, "a", joinPath("", "a"))
	require.Equal(t, "a.b", joinPath("a", "b"))
	require.Equal(t, "a.", joinPath("a", ""))
}
