package counter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCounter(t *testing.T) {

	counter := NewCounter()

	counter.Add("first")
	require.Equal(t, 1, counter.Get("first"))

	counter.Add("second")
	require.Equal(t, 1, counter.Get("second"))

	counter.Add("first")
	require.Equal(t, 2, counter.Get("first"))

	counter.Add("first")
	require.Equal(t, 3, counter.Get("first"))

	counter.Add("second")
	require.Equal(t, 2, counter.Get("second"))

}
