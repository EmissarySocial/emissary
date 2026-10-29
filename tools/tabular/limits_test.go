package tabular

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimits_WithDefaults(t *testing.T) {

	require.Equal(t, DefaultLimits(), Limits{}.withDefaults())
	require.Equal(t, DefaultLimits(), Limits{MaxRows: -1, MaxBytes: -5}.withDefaults())

	custom := Limits{MaxBytes: 1, MaxRows: 2, MaxColumns: 3, MaxCellBytes: 4, MaxDepth: 5, MaxProblems: 6}
	require.Equal(t, custom, custom.withDefaults())

	mixed := Limits{MaxRows: 7}.withDefaults()
	require.Equal(t, 7, mixed.MaxRows)
	require.Equal(t, DefaultMaxBytes, mixed.MaxBytes)
}
