package tabular

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRow_Get(t *testing.T) {

	row := Row{Cells: []Cell{{"token", "a"}, {"summary", nil}}}

	value, ok := row.Get("token")
	require.True(t, ok)
	require.Equal(t, "a", value)

	value, ok = row.Get("summary")
	require.True(t, ok, "a nil value is still present")
	require.Nil(t, value)

	_, ok = row.Get("label")
	require.False(t, ok)

	_, ok = Row{}.Get("token")
	require.False(t, ok)
}
