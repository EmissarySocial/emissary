package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLast(t *testing.T) {
	require.Equal(t, "three", Last("one++two++three", "++"))
}

func TestSplit(t *testing.T) {
	head, tail := Split("one++two++three", "++")
	require.Equal(t, "one", head)
	require.Equal(t, "two++three", tail)
}

func TestSplitTailSimple(t *testing.T) {
	head, tail := SplitTail("one+two+three", "+")
	require.Equal(t, "one+two", head)
	require.Equal(t, "three", tail)
}

func TestSplitTail(t *testing.T) {
	head, tail := SplitTail("one++two++three", "++")
	require.Equal(t, "one++two", head)
	require.Equal(t, "three", tail)
}
