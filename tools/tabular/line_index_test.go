package tabular

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLineIndex(t *testing.T) {

	index := newLineIndex([]byte("ab\ncd\n\nef"))

	require.Equal(t, 1, index.lineAt(0))
	require.Equal(t, 1, index.lineAt(2), "the newline itself ends line 1")
	require.Equal(t, 2, index.lineAt(3))
	require.Equal(t, 3, index.lineAt(6))
	require.Equal(t, 4, index.lineAt(7))
	require.Equal(t, 4, index.lineAt(100))
	require.Equal(t, 1, newLineIndex(nil).lineAt(0))
}
